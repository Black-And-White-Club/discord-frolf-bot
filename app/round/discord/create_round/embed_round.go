package createround

import (
	"context"
	"fmt"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) SendRoundEventEmbed(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (CreateRoundOperationResult, error) {
	return crm.operationWrapper(context.Background(), "SendRoundEventEmbed", func(ctx context.Context) (CreateRoundOperationResult, error) {
		timeValue := time.Time(startTime)
		unixTimestamp := timeValue.Unix()

		user, err := crm.session.User(string(creatorID))
		if err != nil {
			return CreateRoundOperationResult{Error: fmt.Errorf("failed to get creator info: %w", err)}, nil
		}

		creatorName := user.Username
		if member, err := crm.session.GuildMember(guildID, string(creatorID)); err == nil && member.Nick != "" {
			creatorName = member.Nick
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("**%s**", string(title)),
			Description: string(description),
			Color:       0xFF0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "📅 Time",
					Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unixTimestamp, unixTimestamp),
				},
				{
					Name:  "📍 Location",
					Value: string(location),
				},
				{
					Name:   "✅ Accepted",
					Value:  "-",
					Inline: true,
				},
				{
					Name:   "❌ Declined",
					Value:  "-",
					Inline: true,
				},
				{
					Name:   "🤔 Tentative",
					Value:  "-",
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Created by %s", creatorName),
			},
			Timestamp: timeValue.Format(time.RFC3339),
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Accept ✅",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_accept|%s", roundID),
					},
					discordgo.Button{
						Label:    "Decline ❌",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_decline|%s", roundID),
					},
					discordgo.Button{
						Label:    "Tentative 🤔",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_tentative|%s", roundID),
					},
					discordgo.Button{
						Label:    "Edit",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("round_edit|%s", roundID),
					},
					discordgo.Button{
						Label:    "Delete 🗑️",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("round_delete|%s", roundID),
					},
				},
			},
		}

		msg, err := crm.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			return CreateRoundOperationResult{Error: fmt.Errorf("failed to send embed message: %w", err)}, nil
		}

		return CreateRoundOperationResult{Success: msg}, nil
	})
}
