package startround

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// TransformRoundToScorecard uses the operation wrapper.
func (m *startRoundManager) TransformRoundToScorecard(ctx context.Context, payload *roundevents.DiscordRoundStartPayload) (StartRoundOperationResult, error) {
	return m.operationWrapper(ctx, "TransformRoundToScorecard", func(ctx context.Context) (StartRoundOperationResult, error) {
		timeValue := time.Time(*payload.StartTime)
		unixTimestamp := timeValue.Unix()

		participantFields := make([]*discordgo.MessageEmbedField, 0, len(payload.Participants))

		for _, participant := range payload.Participants {
			user, err := m.session.User(string(participant.UserID))
			if err != nil {
				m.logger.ErrorContext(ctx, "Failed to get participant info", attr.Error(err), attr.String("user_id", string(participant.UserID)))
				continue
			}

			username := user.Username
			if member, err := m.session.GuildMember(m.config.Discord.GuildID, string(participant.UserID)); err == nil && member.Nick != "" {
				username = member.Nick
			}

			scoreDisplay := "Score: --"
			if participant.Score != nil {
				scoreDisplay = fmt.Sprintf("Score: %+d", *participant.Score)
			}

			participantFields = append(participantFields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("🏌️ %s", username),
				Value:  scoreDisplay,
				Inline: true,
			})
		}

		var locationStr string
		if payload.Location != nil {
			locationStr = string(*payload.Location)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("**%s** - Round Started", payload.Title),
			Description: fmt.Sprintf("Round at %s has started!", locationStr),
			Color:       0x00AA00,
			Fields: append([]*discordgo.MessageEmbedField{
				{
					Name:  "📅 Started",
					Value: fmt.Sprintf("<t:%d:f>", unixTimestamp),
				},
				{
					Name:  "📍 Location",
					Value: locationStr,
				},
			}, participantFields...),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Round in progress. Use the buttons below to join or record your score.",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Enter Score",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("round_enter_score|%s", payload.RoundID),
						Emoji: &discordgo.ComponentEmoji{
							Name: "💰",
						},
					},
					discordgo.Button{
						Label:    "Join Round LATE",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_join_late|%s", payload.RoundID),
						Emoji: &discordgo.ComponentEmoji{
							Name: "🦇",
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
