package startround

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// --- Constants for Embed Structure and Formatting ---
const (
	// Embed field names
	fieldNameStarted   = "📅 Started"
	fieldNameLocation  = "📍 Location"
	fieldNameAccepted  = "✅ Accepted"
	fieldNameTentative = "🤔 Tentative"
	// fieldNameLeft      = "🚶 Left"

	// Placeholder text used in embed fields
	placeholderNoParticipants  = "*No participants*"
	placeholderUnknownLocation = "Unknown Location"

	// Participant line formatting elements
	// participantIcon = "🏌️" // Removed as it's not desired in the scorecard view
	scorePrefix = "Score:"
	scoreNoData = "--"
	tagPrefix   = "Tag:" // Added constant for tag prefix

	// CustomIDs for message components (buttons)
	customIDEnterScore = "round_enter_score"
	customIDJoinLate   = "round_join_late"

	// Emojis for buttons
	emojiEnterScore = "💰"
	emojiJoinLate   = "🦇"

	// Embed Colors
	colorRoundStarted = 0x00AA00 // Green
)

// --- Helper Struct and Function for Participant Merging ---

// ParticipantData holds simplified participant information for merging.
type ParticipantData struct {
	UserID    sharedtypes.DiscordID
	Response  roundtypes.Response
	Score     *sharedtypes.Score
	TagNumber *sharedtypes.TagNumber // Correct: Includes TagNumber
}

// participantLineRegex extracts User ID from <@ID>, and optionally " Tag: N" and " — Score: +/-N".
// It captures:
// 1: User ID from <@ID> or <@!ID>
// 2: Tag Number from " Tag: N" (if present)
// 3: Score string from " — Score: +/-N" or " — Score: --" (if present)
// Note: The regex is updated to expect " Tag: " with a space before "Tag:".
// Also updated score capture to start after " — Score: ".
var participantLineRegex = regexp.MustCompile(`<@!?(\d+)>` + // Capture User ID
	`(?:` + `\s+` + tagPrefix + `\s*(\d+))?.*?` + // Optionally capture Tag Number, expecting space before Tag:
	`(?:—\s*` + scorePrefix + `\s*([^—]*))?`) // Optionally capture Score string

// parseParticipantLine attempts to extract UserID, Score, and TagNumber from an embed line string.
// It relies on the presence of a Discord user mention (<@USER_ID>) for UserID extraction.
// It attempts to parse " Tag: N" and "— Score: +/-N".
// Returns UserID, Score (*sharedtypes.Score), TagNumber (*sharedtypes.TagNumber), and success boolean.
func parseParticipantLine(ctx context.Context, m *startRoundManager, line string) (sharedtypes.DiscordID, *sharedtypes.Score, *sharedtypes.TagNumber, bool) {
	match := participantLineRegex.FindStringSubmatch(line)
	if len(match) < 2 || match[1] == "" {
		// Must find a User ID mention
		return "", nil, nil, false
	}
	userID := sharedtypes.DiscordID(match[1])

	var tagNumber *sharedtypes.TagNumber // Initialize tagNumber as nil
	if len(match) > 2 && match[2] != "" {
		// Found and captured a tag number
		parsedTag, err := parseInt(match[2]) // Assuming parseInt helper exists
		if err == nil {
			typedTag := sharedtypes.TagNumber(parsedTag)
			tagNumber = &typedTag
		} else {
			// Log a warning if tag number was found but couldn't be parsed
			m.logger.WarnContext(ctx, "Could not parse tag number from line",
				attr.String("tag_str", match[2]),
				attr.String("line", line),
				attr.String("user_id", string(userID)), // Include user ID if parsed
			)
		}
	}

	var score *sharedtypes.Score // Initialize score as nil
	if len(match) > 3 && match[3] != "" {
		// Found and captured a score string
		scoreStr := strings.TrimSpace(match[3])
		if scoreStr != scoreNoData {
			parsedScore, err := parseInt(scoreStr) // Assuming parseInt helper exists
			if err == nil {
				typedScore := sharedtypes.Score(parsedScore)
				score = &typedScore
			} else {
				// Log a warning if score string was found but couldn't be parsed
				m.logger.WarnContext(ctx, "Could not parse score from line",
					attr.String("score_str", scoreStr),
					attr.String("line", line),
					attr.String("user_id", string(userID)), // Include user ID if parsed
				)
			}
		}
	}

	return userID, score, tagNumber, true
}

// Helper function to parse an integer string (assuming it can handle signs +/-)
// This should ideally be in a shared utilities package.
func parseInt(s string) (int, error) {
	var result int
	// Try parsing as signed int first, then unsigned
	_, err := fmt.Sscanf(s, "%+d", &result) // %+d handles both + and - signs, and plain numbers
	if err != nil {
		_, err = fmt.Sscanf(s, "%d", &result)
	}
	return result, err
}

// --- Main Transformation Function ---

// TransformRoundToScorecard transforms the round data into a scorecard embed and components,
// merging participant status from the existing embed with the new payload data.
func (m *startRoundManager) TransformRoundToScorecard(ctx context.Context, payload *roundevents.DiscordRoundStartPayload, existingEmbed *discordgo.MessageEmbed) (StartRoundOperationResult, error) {
	return m.operationWrapper(ctx, "TransformRoundToScorecard", func(ctx context.Context) (StartRoundOperationResult, error) {
		if payload.StartTime == nil {
			err := errors.New("payload StartTime is nil")
			m.logger.ErrorContext(ctx, "TransformRoundToScorecard received nil StartTime in payload")
			return StartRoundOperationResult{Error: err}, fmt.Errorf("invalid payload: StartTime is nil")
		}
		timeValue := time.Time(*payload.StartTime)
		unixTimestamp := timeValue.Unix()

		// Map to hold unique participants by UserID, merging data from existing embed and payload.
		participantsMap := make(map[sharedtypes.DiscordID]*ParticipantData)

		// 1. Populate participantsMap from the existing embed
		if existingEmbed != nil {
			for _, field := range existingEmbed.Fields {
				status := ""
				fieldNameLower := strings.ToLower(field.Name)
				if strings.Contains(fieldNameLower, "accepted") {
					status = string(roundtypes.ResponseAccept)
				} else if strings.Contains(fieldNameLower, "tentative") {
					status = string(roundtypes.ResponseTentative)
				}
				// Add checks for other status field names

				if status != "" && strings.TrimSpace(field.Value) != "" && field.Value != placeholderNoParticipants {
					lines := strings.Split(field.Value, "\n")
					for _, line := range lines {
						// Pass ctx and m to parseParticipantLine for logging
						userID, score, tagNumber, ok := parseParticipantLine(ctx, m, line)
						if ok {
							// Add or update participant from existing embed.
							participantsMap[userID] = &ParticipantData{
								UserID:    userID,
								Response:  roundtypes.Response(status),
								Score:     score,     // Keep existing score if available
								TagNumber: tagNumber, // Keep existing tag number parsed from embed
							}
						} else {
							m.logger.WarnContext(ctx, "Could not parse participant line from existing embed field",
								attr.String("field_name", field.Name), attr.String("line", line))
						}
					}
				}
			}
		}

		// 2. Update/add participants from the payload (payload data takes precedence)
		for _, p := range payload.Participants {
			// Overwrite or add participant data with information from the payload.
			// This ensures the latest status, score, and tag number from the payload are used.
			// Note: If payload.TagNumber is nil or 0, it will overwrite a parsed tag number from the embed.
			participantsMap[p.UserID] = &ParticipantData{
				UserID:    p.UserID,
				Response:  p.Response,  // Use status from payload
				Score:     p.Score,     // Use score from payload (can be nil)
				TagNumber: p.TagNumber, // Use tag number from payload
			}
		}

		// 3. Build acceptedLines and tentativeLines from the merged participantsMap
		acceptedLines := []string{}
		tentativeLines := []string{}
		// leftLines := []string{} // Add slices for other statuses

		// Convert map to slice for consistent ordering
		participants := make([]*ParticipantData, 0, len(participantsMap))
		for _, p := range participantsMap {
			participants = append(participants, p)
		}

		// Sort participants by tag number (if present), then by user ID for consistent ordering
		sort.Slice(participants, func(i, j int) bool {
			// If both have tag numbers, sort by tag number
			if participants[i].TagNumber != nil && participants[j].TagNumber != nil {
				return *participants[i].TagNumber < *participants[j].TagNumber
			}
			// If only one has a tag number, that one comes first
			if participants[i].TagNumber != nil && participants[j].TagNumber == nil {
				return true
			}
			if participants[i].TagNumber == nil && participants[j].TagNumber != nil {
				return false
			}
			// If neither has a tag number, sort by user ID
			return string(participants[i].UserID) < string(participants[j].UserID)
		})

		for _, p := range participants {
			// Add debug log here to inspect participant data before formatting
			m.logger.DebugContext(ctx, "Formatting participant line",
				attr.String("user_id", string(p.UserID)),
				attr.String("status", string(p.Response)),
				attr.Any("score", p.Score),          // Use attr.Any for pointer/nil values
				attr.Any("tag_number", p.TagNumber), // Use attr.Any for pointer/nil values
			)

			// Fetch user/member info (optional for this format, but helpful for logging)
			// user, err := m.session.User(string(p.UserID))
			// if err != nil { ... log error ... }

			// Format the tag number part " Tag: N" if available
			tagDisplayPart := ""
			if p.TagNumber != nil && *p.TagNumber > 0 {
				tagDisplayPart = fmt.Sprintf(" %s %d", tagPrefix, *p.TagNumber)
			}

			// Format the score display part " — Score: +/-N" or " — Score: --"
			scoreDisplayPart := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
			if p.Score != nil { // Use the score from the merged ParticipantData
				scoreDisplayPart = fmt.Sprintf(" — %s %+d", scorePrefix, *p.Score)
			}

			// Format the complete participant line using the desired scorecard format: <@USER_ID> Tag: N — Score: +/-N
			// Assemble parts: Mention + Tag Part + Score Part
			line := fmt.Sprintf("<@%s>%s%s", p.UserID, tagDisplayPart, scoreDisplayPart)

			// Append the formatted line to the correct status slice
			switch p.Response {
			case roundtypes.ResponseAccept:
				acceptedLines = append(acceptedLines, line)
			case roundtypes.ResponseTentative:
				tentativeLines = append(tentativeLines, line)
			// case roundtypes.ResponseLeft:
			// 	leftLines = append(leftLines, line)
			default:
				m.logger.WarnContext(ctx, "Participant with unhandled response status encountered",
					attr.String("user_id", string(p.UserID)), attr.String("response", string(p.Response)))
			}
		}

		// Determine location string, prioritizing payload, then existing embed
		var locationStr string
		if payload.Location != nil {
			locationStr = string(*payload.Location)
		} else if existingEmbed != nil {
			for _, field := range existingEmbed.Fields {
				if strings.Contains(strings.ToLower(field.Name), "location") {
					locationStr = field.Value
					break
				}
			}
		}
		if locationStr == "" {
			locationStr = placeholderUnknownLocation
		}

		// Define the base embed fields
		embedFields := []*discordgo.MessageEmbedField{
			{
				Name:  fieldNameStarted,
				Value: fmt.Sprintf("<t:%d:f>", unixTimestamp),
			},
			{
				Name:  fieldNameLocation,
				Value: locationStr,
			},
		}

		// Only add status fields if there are participants
		if len(participantsMap) > 0 {
			embedFields = append(embedFields,
				&discordgo.MessageEmbedField{
					Name:   fieldNameAccepted,
					Value:  placeholderNoParticipants,
					Inline: false,
				},
				&discordgo.MessageEmbedField{
					Name:   fieldNameTentative,
					Value:  placeholderNoParticipants,
					Inline: false,
				},
			)

			// Populate the status fields if lists are not empty
			if len(acceptedLines) > 0 {
				embedFields[2].Value = strings.Join(acceptedLines, "\n")
			}
			if len(tentativeLines) > 0 {
				embedFields[3].Value = strings.Join(tentativeLines, "\n")
			}
		}
		// Populate other status fields if applicable (adjust index)

		// Construct the final embed
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("**%s** - Round Started", payload.Title),
			Description: fmt.Sprintf("Round at %s has started!", locationStr),
			Color:       colorRoundStarted,
			Fields:      embedFields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Round in progress. Use the buttons below to join or record your score.",
			},
			Timestamp: time.Now().Format(time.RFC3339), // Use current time for "updated at"
		}

		// Define the components (buttons)
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Enter Score",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("%s|%s", customIDEnterScore, payload.RoundID),
						Emoji: &discordgo.ComponentEmoji{
							Name: emojiEnterScore,
						},
					},
					discordgo.Button{
						Label:    "Join Round LATE",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("%s|%s", customIDJoinLate, payload.RoundID),
						Emoji: &discordgo.ComponentEmoji{
							Name: emojiJoinLate,
						},
					},
				},
			},
		}

		return StartRoundOperationResult{
			Success: struct {
				Embed      *discordgo.MessageEmbed
				Components []discordgo.MessageComponent
			}{
				Embed:      embed,
				Components: components,
			},
		}, nil
	})
}
