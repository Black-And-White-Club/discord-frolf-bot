package finalizeround

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// TransformRoundToFinalizedScorecard transforms the round event embed into a finalized scorecard format
// showing participants with their final scores and modifying the UI to indicate the round is finalized
func (frm *finalizeRoundManager) TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	// Convert Start Time to Unix Timestamp
	unixTimestamp := time.Time(*payload.StartTime).Unix()

	// Create participant fields
	participantFields := make([]*discordgo.MessageEmbedField, 0, len(payload.Participants))

	for _, participant := range payload.Participants {
		user, err := frm.session.User(string(participant.UserID))
		if err != nil {
			frm.logger.Error(context.TODO(), "Failed to get participant info",
				attr.Error(err), attr.String("user_id", string(participant.UserID)))
			continue
		}

		username := user.Username
		if member, err := frm.session.GuildMember(frm.config.Discord.GuildID, string(participant.UserID)); err == nil && member.Nick != "" {
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

	locationStr := ""
	if payload.Location != nil {
		locationStr = string(*payload.Location)
	}

	// Embed Fields
	embedFields := []*discordgo.MessageEmbedField{
		{Name: "📅 Started", Value: fmt.Sprintf("<t:%d:f>", unixTimestamp)},
		{Name: "📍 Location", Value: locationStr},
	}

	// Add participant fields
	embedFields = append(embedFields, participantFields...)

	// Construct the embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("**%s** - Round Finalized", payload.Title),
		Description: fmt.Sprintf("Round at %s has been finalized. Admin/Editor access required for score updates.", locationStr),
		Color:       0x0000FF, // Blue for finalized round
		Fields:      embedFields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Round has been finalized. Only admins/editors can update scores.",
		},
		Timestamp: time.Now().Format(time.RFC3339), // Current time when finalized
	}

	// Keep the same button but with modified text to indicate admin requirement
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Admin/Editor Score Update",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("round_enter_score_finalized|round-%d", payload.RoundID),
					Emoji:    &discordgo.ComponentEmoji{Name: "🔒"},
				},
			},
		},
	}

	return embed, components, nil
}
