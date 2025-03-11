package createround

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) SendRoundEventEmbed(channelID, eventID, title, description string, startTime time.Time, location, creatorID string) (*discordgo.Message, error) {
	// Convert time to Unix for Discord formatting
	unixTimestamp := startTime.Unix()

	// Fetch the user's information from the Discord API
	user, err := crm.session.User(creatorID)
	if err != nil {
		return nil, err
	}

	// Get the user's name on the server
	guildID := crm.config.Discord.GuildID
	member, err := crm.session.GuildMember(guildID, creatorID)
	if err != nil {
		creatorName := user.Username
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("**%s**", title),
			Description: description,
			Color:       0xFF0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "📅 Time",
					Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unixTimestamp, unixTimestamp),
				},
				{
					Name:  "📍 Location",
					Value: location,
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
			Timestamp: startTime.Format(time.RFC3339),
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Accept ✅",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_accept|%s", eventID),
					},
					discordgo.Button{
						Label:    "Decline ❌",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_decline|%s", eventID),
					},
					discordgo.Button{
						Label:    "Tentative 🤔",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("round_tentative|%s", eventID),
					},
				},
			},
		}

		messageSend := &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		}

		return crm.session.ChannelMessageSendComplex(channelID, messageSend)
	}

	// Use the member's nickname if available, otherwise use the user's username
	creatorName := member.Nick
	if creatorName == "" {
		creatorName = user.Username
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("**%s**", title),
		Description: description,
		Color:       0xFF0000,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "📅 Time",
				Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unixTimestamp, unixTimestamp),
			},
			{
				Name:  "📍 Location",
				Value: location,
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
		Timestamp: startTime.Format(time.RFC3339),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Accept ✅",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_accept|%s", eventID),
				},
				discordgo.Button{
					Label:    "Decline ❌",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_decline|%s", eventID),
				},
				discordgo.Button{
					Label:    "Tentative 🤔",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("round_tentative|%s", eventID),
				},
			},
		},
	}

	messageSend := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}

	return crm.session.ChannelMessageSendComplex(channelID, messageSend)
}

func (crm *createRoundManager) HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate, response string) {
	user := i.Member.User
	eventID := strings.Split(i.MessageComponentData().CustomID, "|")[1]

	slog.Info("Processing round RSVP", attr.String("user", user.Username), attr.String("response", response), attr.String("event_id", eventID))

	// Acknowledge the interaction (so Discord doesn't time out)
	err := crm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		slog.Error("Failed to acknowledge interaction", attr.Error(err))
	}

	crm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("%s has %s the event!", user.Mention(), response),
	})
}

func (crm *createRoundManager) UpdateRoundEventEmbed(channelID, messageID string, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.RoundParticipant) error {
	// Format the participant lists
	accepted := formatParticipants(acceptedParticipants)
	declined := formatParticipants(declinedParticipants)
	tentative := formatParticipants(tentativeParticipants)

	// Create the updated embed message
	embed := &discordgo.MessageEmbed{
		Title:       "Updated Round Event", // You can customize this title
		Description: "Updated RSVP Status",
		Color:       0x00ff00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Accepted",
				Value:  accepted,
				Inline: true,
			},
			{
				Name:   "Declined",
				Value:  declined,
				Inline: true,
			},
			{
				Name:   "Tentative",
				Value:  tentative,
				Inline: true,
			},
		},
	}

	// Update the message in the channel
	_, err := crm.session.ChannelMessageEditEmbed(channelID, messageID, embed)
	return err
}

func formatParticipants(participants []roundtypes.RoundParticipant) string {
	if len(participants) == 0 {
		return "-"
	}

	var names []string
	for _, participant := range participants {
		names = append(names, fmt.Sprintf("%s (Tag: %d)", participant.DiscordID, participant.TagNumber))
	}
	return strings.Join(names, "\n")
}
