package startround

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// TransformRoundToScorecard transforms the round event embed into a scorecard format
// showing participants with placeholder for scores
func (srm *startRoundManager) TransformRoundToScorecard(payload *roundevents.DiscordRoundStartPayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	// Convert time for Discord formatting
	timeValue := time.Time(*payload.StartTime)
	unixTimestamp := timeValue.Unix()

	// Create participant fields
	participantFields := make([]*discordgo.MessageEmbedField, 0, len(payload.Participants))

	// Add field for each participant
	for _, participant := range payload.Participants {
		// Get the participant's display name
		user, err := srm.session.User(string(participant.UserID))
		if err != nil {
			srm.logger.Error(context.TODO(), "Failed to get participant info", attr.Error(err), attr.String("user_id", string(participant.UserID)))
			continue
		}

		username := user.Username
		if member, err := srm.session.GuildMember("guild-123", string(participant.UserID)); err == nil && member.Nick != "" {
			username = member.Nick
		}

		// Determine score display
		scoreDisplay := "Score: --"
		if participant.Score != nil {
			scoreDisplay = fmt.Sprintf("Score: %+d", *participant.Score)
		}

		// Add field for this participant with score
		participantFields = append(participantFields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🏌️ %s", username),
			Value:  scoreDisplay,
			Inline: true,
		})
	}

	// Get location string safely
	var locationStr string
	if payload.Location != nil {
		locationStr = string(*payload.Location)
	}

	// Construct the embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("**%s** - Round Started", payload.Title),
		Description: fmt.Sprintf("Round at %s has started!", locationStr),
		Color:       0x00AA00, // Green for active round
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

	// Create components for score entry and joining
	// Using a single action row with two buttons for best mobile compatibility
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Enter Score",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("round_enter_score|round-%d", payload.RoundID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "Enter Score",
						ID:   "💰",
					},
				},
				discordgo.Button{
					Label:    "Join Round LATE",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_join_late|round-%d", payload.RoundID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "Join Round LATE",
						ID:   "🦇",
					},
				},
			},
		},
	}

	return embed, components, nil
}
