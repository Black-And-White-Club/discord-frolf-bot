package startround

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

const (
	fieldNameStarted   = "📅 Started"
	fieldNameLocation  = "📍 Location"
	fieldNameAccepted  = "✅ Accepted"
	fieldNameTentative = "🤔 Tentative"

	placeholderNoParticipants  = "*No participants*"
	placeholderUnknownLocation = "Unknown Location"

	scorePrefix = "Score:"
	scoreNoData = "--"
	tagPrefix   = "Tag:"

	customIDEnterScore      = "round_enter_score"
	customIDJoinLate        = "round_join_late"
	customIDUploadScorecard = "round_upload_scorecard"

	emojiEnterScore      = "💰"
	emojiJoinLate        = "🏃"
	emojiUploadScorecard = "📋"

	colorRoundStarted = 0x00AA00
)

type ParticipantData struct {
	UserID    sharedtypes.DiscordID
	Response  roundtypes.Response
	Score     *sharedtypes.Score
	TagNumber *sharedtypes.TagNumber
}

var participantLineRegex = regexp.MustCompile(
	`<@!?(\d+)>(?:\s+Tag:\s*(\d+))?(?:\s*—\s*Score:\s*([^—]+))?`,
)

func parseParticipantLine(
	ctx context.Context,
	m *startRoundManager,
	line string,
) (sharedtypes.DiscordID, *sharedtypes.Score, *sharedtypes.TagNumber, bool) {

	match := participantLineRegex.FindStringSubmatch(line)
	if len(match) < 2 {
		return "", nil, nil, false
	}

	userID := sharedtypes.DiscordID(match[1])

	var tagNumber *sharedtypes.TagNumber
	if match[2] != "" {
		if v, err := strconv.Atoi(match[2]); err == nil {
			t := sharedtypes.TagNumber(v)
			tagNumber = &t
		} else {
			m.logger.WarnContext(ctx, "Failed to parse tag number",
				attr.String("value", match[2]),
				attr.String("line", line),
			)
		}
	}

	var score *sharedtypes.Score
	if match[3] != "" && strings.TrimSpace(match[3]) != scoreNoData {
		if v, err := strconv.Atoi(strings.TrimSpace(match[3])); err == nil {
			s := sharedtypes.Score(v)
			score = &s
		} else {
			m.logger.WarnContext(ctx, "Failed to parse score",
				attr.String("value", match[3]),
				attr.String("line", line),
			)
		}
	}

	return userID, score, tagNumber, true
}

func responseFromFieldName(name string) roundtypes.Response {
	l := strings.ToLower(name)
	switch {
	case strings.Contains(l, "accepted"):
		return roundtypes.ResponseAccept
	case strings.Contains(l, "tentative"):
		return roundtypes.ResponseTentative
	default:
		return ""
	}
}

func formatParticipantLine(p *ParticipantData) string {
	tagPart := ""
	if p.TagNumber != nil && *p.TagNumber > 0 {
		tagPart = fmt.Sprintf(" %s %d", tagPrefix, *p.TagNumber)
	}

	scorePart := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
	if p.Score != nil {
		scorePart = fmt.Sprintf(" — %s %+d", scorePrefix, *p.Score)
	}

	return fmt.Sprintf("<@%s>%s%s", p.UserID, tagPart, scorePart)
}

func (m *startRoundManager) TransformRoundToScorecard(
	ctx context.Context,
	payload *roundevents.DiscordRoundStartPayloadV1,
	existingEmbed *discordgo.MessageEmbed,
) (StartRoundOperationResult, error) {

	return m.operationWrapper(ctx, "TransformRoundToScorecard", func(ctx context.Context) (StartRoundOperationResult, error) {
		if payload.StartTime == nil {
			err := errors.New("payload StartTime is nil")
			m.logger.ErrorContext(ctx, err.Error())
			return StartRoundOperationResult{Error: err}, err
		}

		startUnix := time.Time(*payload.StartTime).Unix()
		participants := make(map[sharedtypes.DiscordID]*ParticipantData)

		if existingEmbed != nil {
			for _, field := range existingEmbed.Fields {
				resp := responseFromFieldName(field.Name)
				if resp == "" || field.Value == placeholderNoParticipants {
					continue
				}

				for _, line := range strings.Split(field.Value, "\n") {
					userID, score, tag, ok := parseParticipantLine(ctx, m, line)
					if !ok {
						continue
					}

					participants[userID] = &ParticipantData{
						UserID:    userID,
						Response:  resp,
						Score:     score,
						TagNumber: tag,
					}
				}
			}
		}

		for _, p := range payload.Participants {
			existing := participants[p.UserID]
			if existing == nil {
				existing = &ParticipantData{UserID: p.UserID}
			}

			existing.Response = p.Response
			if p.Score != nil {
				existing.Score = p.Score
			}
			if p.TagNumber != nil {
				existing.TagNumber = p.TagNumber
			}

			participants[p.UserID] = existing
		}

		list := make([]*ParticipantData, 0, len(participants))
		for _, p := range participants {
			list = append(list, p)
		}

		sort.Slice(list, func(i, j int) bool {
			a, b := list[i], list[j]
			if a.TagNumber != nil && b.TagNumber != nil {
				return *a.TagNumber < *b.TagNumber
			}
			if a.TagNumber != nil {
				return true
			}
			if b.TagNumber != nil {
				return false
			}
			return string(a.UserID) < string(b.UserID)
		})

		var accepted, tentative []string
		for _, p := range list {
			line := formatParticipantLine(p)
			switch p.Response {
			case roundtypes.ResponseAccept:
				accepted = append(accepted, line)
			case roundtypes.ResponseTentative:
				tentative = append(tentative, line)
			}
		}

		location := placeholderUnknownLocation
		if payload.Location != "" {
			location = string(payload.Location)
		}

		fields := []*discordgo.MessageEmbedField{
			{Name: fieldNameStarted, Value: fmt.Sprintf("<t:%d:f>", startUnix)},
			{Name: fieldNameLocation, Value: location},
			{Name: fieldNameAccepted, Value: placeholderNoParticipants},
			{Name: fieldNameTentative, Value: placeholderNoParticipants},
		}

		if len(accepted) > 0 {
			fields[2].Value = strings.Join(accepted, "\n")
		}
		if len(tentative) > 0 {
			fields[3].Value = strings.Join(tentative, "\n")
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("**%s** - Round Started", payload.Title),
			Description: fmt.Sprintf("Round at %s has started!", location),
			Color:       colorRoundStarted,
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Round in progress. Use the buttons below to join or record your score.",
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Enter Score",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("%s|%s", customIDEnterScore, payload.RoundID),
						Emoji:    &discordgo.ComponentEmoji{Name: emojiEnterScore},
					},
					discordgo.Button{
						Label:    "Join Round LATE",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("%s|%s", customIDJoinLate, payload.RoundID),
						Emoji:    &discordgo.ComponentEmoji{Name: emojiJoinLate},
					},
					discordgo.Button{
						Label:    "Upload Scorecard",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("%s|%s", customIDUploadScorecard, payload.RoundID),
						Emoji:    &discordgo.ComponentEmoji{Name: emojiUploadScorecard},
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
