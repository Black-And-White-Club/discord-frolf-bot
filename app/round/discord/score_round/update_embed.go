package scoreround

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord" // Import roundtypes for Response
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// --- Constants for Embed Structure and Formatting (Copied/Adapted from startround) ---
const (

	// Placeholder text used in embed fields (Should match startround)
	placeholderNoParticipants = "*No participants*"

	// Participant line formatting elements (Should match startround)
	scorePrefix = "Score:"
	scoreNoData = "--"
	tagPrefix   = "Tag:"
)

// --- Helper Struct and Function for Participant Data within Embed (Copied/Adapted from startround) ---

// ParticipantDataFromEmbed holds participant information parsed from an embed line.
type ParticipantDataFromEmbed struct {
	UserID    sharedtypes.DiscordID
	TagNumber *sharedtypes.TagNumber
	Score     *sharedtypes.Score
	LineText  string // Store the original line text for easier reconstruction
}

// participantLineRegex extracts User ID from <@ID>, and optionally " Tag: N" and " — Score: +/-N".
// It captures:
// 1: User ID from <@ID> or <@!ID>
// 2: Tag Number from " Tag: N" (if present)
// 3: Score string from " — Score: +/-N" or " — Score: --" (if present)
// participantLineRegex extracts User ID from <@ID>, and optionally " Tag: N" and " — Score: +/-N".
// Updated to handle alphanumeric user IDs and different dash characters
var participantLineRegex = regexp.MustCompile(`<@!?([a-zA-Z0-9]+)>` + // Capture User ID (allow alphanumeric for tests)
	`(?:\s+` + tagPrefix + `\s*(\d+))?` + // Optionally capture Tag Number with flexible whitespace
	`(?:\s*[—–-]\s*` + scorePrefix + `\s*([+\-]?\d+|` + scoreNoData + `))?`) // Optionally capture Score string, handle different dash types
// parseParticipantLine attempts to extract UserID, Score, and TagNumber from an embed line string.
// It relies on the presence of a Discord user mention (<@USER_ID>) for UserID extraction.
// It attempts to parse " Tag: N" and "— Score: +/-N".
// Returns UserID, Score (*sharedtypes.Score), TagNumber (*sharedtypes.TagNumber), and success boolean.
// Copied from startround, added context and logger for internal logging
// parseParticipantLine attempts to extract UserID, Score, and TagNumber from an embed line string.
func (srm *scoreRoundManager) parseParticipantLine(ctx context.Context, line string) (sharedtypes.DiscordID, *sharedtypes.Score, *sharedtypes.TagNumber, bool) {
	srm.logger.DebugContext(ctx, "Parsing participant line",
		attr.String("line", line),
		attr.String("line_bytes", fmt.Sprintf("%#v", line)), // Show actual bytes
	)

	match := participantLineRegex.FindStringSubmatch(line)
	srm.logger.DebugContext(ctx, "Regex match result",
		attr.Any("matches", match),
		attr.Int("match_count", len(match)),
		attr.String("regex_pattern", participantLineRegex.String()),
	)

	if len(match) < 2 || match[1] == "" {
		// Must find a User ID mention
		srm.logger.DebugContext(ctx, "No user ID found in line",
			attr.String("line", line),
			attr.String("regex_pattern", participantLineRegex.String()),
		)
		return "", nil, nil, false
	}
	userID := sharedtypes.DiscordID(match[1])

	var tagNumber *sharedtypes.TagNumber // Initialize tagNumber as nil
	if len(match) > 2 && match[2] != "" {
		// Found and captured a tag number
		parsedTag, err := parseInt(match[2])
		if err == nil {
			typedTag := sharedtypes.TagNumber(parsedTag)
			tagNumber = &typedTag
			srm.logger.DebugContext(ctx, "Parsed tag number",
				attr.String("tag_str", match[2]),
				attr.Int("tag_value", parsedTag),
			)
		} else {
			srm.logger.WarnContext(ctx, "Could not parse tag number from line in UpdateScoreEmbed",
				attr.String("tag_str", match[2]),
				attr.String("line", line),
				attr.String("user_id_attempt", string(userID)),
				attr.Error(err),
			)
		}
	}

	var score *sharedtypes.Score // Initialize score as nil
	if len(match) > 3 && match[3] != "" {
		// Found and captured a score string
		scoreStr := strings.TrimSpace(match[3])
		srm.logger.DebugContext(ctx, "Found score string", attr.String("score_str", scoreStr))

		if scoreStr != scoreNoData {
			parsedScore, err := parseInt(scoreStr)
			if err == nil {
				typedScore := sharedtypes.Score(parsedScore)
				score = &typedScore
				srm.logger.DebugContext(ctx, "Parsed score",
					attr.String("score_str", scoreStr),
					attr.Int("score_value", parsedScore),
				)
			} else {
				srm.logger.WarnContext(ctx, "Could not parse score from line in UpdateScoreEmbed",
					attr.String("score_str", scoreStr),
					attr.String("line", line),
					attr.String("user_id_attempt", string(userID)),
					attr.Error(err),
				)
			}
		}
	}

	srm.logger.DebugContext(ctx, "Parsed participant line successfully",
		attr.String("user_id", string(userID)),
		attr.Any("tag_number", tagNumber),
		attr.Any("score", score),
	)

	return userID, score, tagNumber, true
}

// Helper function to parse an integer string (assuming it can handle signs +/-)
// Copied from startround
func parseInt(s string) (int, error) {
	var result int
	// Try parsing as signed int first, then unsigned
	_, err := fmt.Sscanf(s, "%+d", &result) // %+d handles both + and - signs, and plain numbers
	if err != nil {
		_, err = fmt.Sscanf(s, "%d", &result)
	}
	return result, err
}

// UpdateScoreEmbed updates the score for a specific participant in the scorecard embed.
// It finds the participant's line by UserID, preserves their existing tag, and updates their score.
// It assumes the embed fields for participants use the format "<@USER_ID> Tag: N — Score: +/-N" (or similar parsable format).
func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_score_embed")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(userID))

	return srm.operationWrapper(ctx, "update_score_embed", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		if srm.session == nil {
			return ScoreRoundOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}

		// 1. Fetch the existing message and embed
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to fetch message for score update",
				attr.Error(err),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
				attr.String("user_id", string(userID)),
			)
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message for score update: %w", err)
		}

		if len(message.Embeds) == 0 {
			srm.logger.WarnContext(ctx, "No embeds found in message for score update",
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
				attr.String("user_id", string(userID)),
			)
			// Consider if this is an error or just means the embed hasn't been created yet
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
		}

		// Assuming the scorecard is the first embed
		embed := message.Embeds[0]
		userFoundAndScoreUpdated := false

		// 2. Iterate through relevant embed fields (Accepted, Tentative, etc.)
		participantFields := []*discordgo.MessageEmbedField{}
		// We need to find the original field index by name to update its value later
		fieldNameMap := map[string]int{}

		for i, field := range embed.Fields {
			fieldNameLower := strings.ToLower(field.Name)
			// Check for participant status fields - be more inclusive
			if strings.Contains(fieldNameLower, "accepted") ||
				strings.Contains(fieldNameLower, "tentative") ||
				strings.Contains(fieldNameLower, "declined") ||
				strings.Contains(fieldNameLower, "participants") ||
				strings.Contains(fieldNameLower, "✅") ||
				strings.Contains(fieldNameLower, "❓") ||
				strings.Contains(fieldNameLower, "❌") {
				participantFields = append(participantFields, field)
				// Store the index of the original field
				fieldNameMap[field.Name] = i

				srm.logger.DebugContext(ctx, "Found participant field for score update",
					attr.String("field_name", field.Name),
					attr.String("field_value", field.Value),
					attr.Int("field_index", i),
				)
			}
		}

		if len(participantFields) == 0 {
			srm.logger.WarnContext(ctx, "No participant fields found in embed for score update",
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
				attr.String("user_id", string(userID)),
			)
			// Consider if this is an error or just means the embed structure is unexpected
			return ScoreRoundOperationResult{Success: "No participant fields found"}, nil
		}

		// 3. Parse lines in participant fields, find the user, update score, and reconstruct field values
		updatedFieldValues := map[string]string{} // Map field name to its new value

		for _, field := range participantFields {
			srm.logger.DebugContext(ctx, "Processing field for score update",
				attr.String("field_name", field.Name),
				attr.String("field_value", field.Value),
				attr.String("target_user_id", string(userID)),
			)

			originalLines := strings.Split(field.Value, "\n")
			newLines := []string{}

			if strings.TrimSpace(field.Value) == "" || field.Value == placeholderNoParticipants {
				// If the field is empty or placeholder, the user isn't in this status list
				updatedFieldValues[field.Name] = field.Value // Keep as is
				srm.logger.DebugContext(ctx, "Field is empty or placeholder, skipping",
					attr.String("field_name", field.Name),
				)
				continue
			}

			for lineIdx, line := range originalLines {
				srm.logger.DebugContext(ctx, "Processing line in field",
					attr.String("field_name", field.Name),
					attr.Int("line_index", lineIdx),
					attr.String("line", line),
				)

				parsedUserID, parsedScore, parsedTagNumber, ok := srm.parseParticipantLine(ctx, line)

				if !ok {
					// If a line can't be parsed, log a warning and keep the original line
					srm.logger.WarnContext(ctx, "Failed to parse participant line in UpdateScoreEmbed",
						attr.String("line", line),
						attr.String("field_name", field.Name),
						attr.String("message_id", messageID),
					)
					newLines = append(newLines, line) // Preserve unparsable lines? Or skip?
					continue
				}

				srm.logger.DebugContext(ctx, "Successfully parsed line",
					attr.String("parsed_user_id", string(parsedUserID)),
					attr.String("target_user_id", string(userID)),
					attr.Bool("is_match", parsedUserID == userID),
				)

				// Check if this line belongs to the user whose score is being updated
				if parsedUserID == userID {
					userFoundAndScoreUpdated = true // Mark that the user was found in any relevant field
					srm.logger.DebugContext(ctx, "Found target user, updating score",
						attr.String("user_id", string(userID)),
						attr.String("field_name", field.Name),
					)

					// Format the updated line with the new score, preserving the tag number
					scoreDisplayPart := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
					if score != nil {
						scoreDisplayPart = fmt.Sprintf(" — %s %+d", scorePrefix, *score)
					}

					tagDisplayPart := ""
					if parsedTagNumber != nil && *parsedTagNumber > 0 {
						tagDisplayPart = fmt.Sprintf(" %s %d", tagPrefix, *parsedTagNumber)
					}

					// The new line format: <@USER_ID> Tag: N — Score: +/-N
					updatedLine := fmt.Sprintf("<@%s>%s%s", userID, tagDisplayPart, scoreDisplayPart)

					newLines = append(newLines, updatedLine) // Add the updated line
					srm.logger.DebugContext(ctx, "Updated participant line in embed field",
						attr.String("user_id", string(userID)),
						attr.String("field_name", field.Name),
						attr.String("original_line", line),
						attr.String("updated_line", updatedLine),
						attr.Any("new_score", score),
					)

				} else {
					// If it's not the user being updated, keep their original line format (by re-parsing and formatting)
					// This ensures consistent formatting and preserves their original data
					scoreDisplayPart := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
					if parsedScore != nil {
						scoreDisplayPart = fmt.Sprintf(" — %s %+d", scorePrefix, *parsedScore) // Use parsed score
					}

					tagDisplayPart := ""
					if parsedTagNumber != nil && *parsedTagNumber > 0 {
						tagDisplayPart = fmt.Sprintf(" %s %d", tagPrefix, *parsedTagNumber) // Use parsed tag
					}

					// Reformat the line using parsed data
					reformattedLine := fmt.Sprintf("<@%s>%s%s", parsedUserID, tagDisplayPart, scoreDisplayPart)

					newLines = append(newLines, reformattedLine) // Add the reformatted line
				}
			}

			// Check if any lines were successfully processed for this field.
			// If the original field had content but all lines failed to parse, newLines will be empty.
			if len(originalLines) > 0 && len(newLines) == 0 {
				srm.logger.ErrorContext(ctx, "Participant field resulted in zero lines after parsing/updating in UpdateScoreEmbed",
					attr.String("field_name", field.Name),
					attr.String("original_value", field.Value),
					attr.String("user_id", string(userID)),
				)
				// Decide how to handle: keep original value on error? This might be best to avoid data loss.
				updatedFieldValues[field.Name] = field.Value // Keep original value on error
			} else if len(newLines) == 0 {
				// If the original field was empty or placeholder and no lines were added (expected)
				updatedFieldValues[field.Name] = placeholderNoParticipants
			} else {
				// Reconstruct the field value from the new lines
				updatedFieldValues[field.Name] = strings.Join(newLines, "\n")
			}

		}

		if !userFoundAndScoreUpdated {
			srm.logger.WarnContext(ctx, "Participant not found in checked embed fields for score update",
				attr.String("user_id", string(userID)),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
			)
			// User not found in the expected participant fields. Decide how to handle this.
			// Maybe they haven't joined or are in a different status field not checked?
			return ScoreRoundOperationResult{Success: fmt.Sprintf("User %s not found in embed fields to update score", userID)}, nil
		}

		// 4. Update the embed with the new field values
		// Iterate through the original embed fields to update them by index
		for i, field := range embed.Fields {
			// Check if this field's name is one we processed and updated
			if newValue, ok := updatedFieldValues[field.Name]; ok {
				// Update the value of the original embed field at its index
				embed.Fields[i].Value = newValue
			}
		}

		// 5. Edit the Discord message with the modified embed
		edit := &discordgo.MessageEdit{
			Channel: channelID,
			ID:      messageID,
		}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed}) // Set the modified embed

		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to update message with new score",
				attr.Error(err),
				attr.String("user_id", string(userID)),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
			)
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to edit message for score update: %w", err)
		}

		scoreValue := 0
		if score != nil {
			scoreValue = int(*score)
		}

		srm.logger.InfoContext(ctx, "Successfully updated user score in embed",
			attr.String("user_id", string(userID)),
			attr.Int("score", scoreValue),
			attr.String("channel_id", channelID),
			attr.String("message_id", messageID),
		)

		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}
