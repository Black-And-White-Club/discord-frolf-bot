package createround

import (
	"fmt"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) SendRoundEventEmbed(channelID string, title roundtypes.Title, description roundtypes.Description, startTime roundtypes.StartTime, location roundtypes.Location, creatorID roundtypes.UserID, roundID roundtypes.ID) (*discordgo.Message, error) {
	// Convert time for Discord formatting
	timeValue := time.Time(startTime)
	unixTimestamp := timeValue.Unix()

	// Get the creator's display name
	user, err := crm.session.User(string(creatorID))
	if err != nil {
		return nil, fmt.Errorf("failed to get creator info: %w", err)
	}
	creatorName := user.Username
	if member, err := crm.session.GuildMember(crm.config.Discord.GuildID, string(creatorID)); err == nil && member.Nick != "" {
		creatorName = member.Nick
	}

	// Construct the embed
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
		Timestamp: time.Time(startTime).Format(time.RFC3339),
	}

	// Construct the buttons
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Accept ✅",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_accept|%d", int64(roundID)),
				},
				discordgo.Button{
					Label:    "Decline ❌",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_decline|%d", int64(roundID)),
				},
				discordgo.Button{
					Label:    "Tentative 🤔",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_tentative|%d", int64(roundID)),
				},
				discordgo.Button{
					Label:    "Delete 🗑️",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("round_delete|%d", int64(roundID)),
				},
			},
		},
	}

	// Send the message
	message, err := crm.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send discord message: %w", err)
	}

	return message, nil
}
