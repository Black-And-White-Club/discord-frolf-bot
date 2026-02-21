package createround

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (crm *createRoundManager) SendRoundEventEmbed(guildID string, channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (CreateRoundOperationResult, error) {
	return crm.operationWrapper(context.Background(), "SendRoundEventEmbed", func(ctx context.Context) (CreateRoundOperationResult, error) {
		// Validate channel type to debug HTTP 405 errors
		channel, err := crm.session.GetChannel(channelID)
		if err != nil {
			// Just log warning, try to proceed anyway in case it's a transient permission issue
			// preventing us from seeing the channel but not posting to it (unlikely but safe)
			// Actually, if we can't see it, we probably can't post to it.
			crm.logger.WarnContext(ctx, "Failed to inspect channel before sending embed", attr.Error(err), attr.String("channel_id", channelID))
		} else {
			crm.logger.InfoContext(ctx, "Inspecting target channel",
				attr.String("channel_id", channel.ID),
				attr.String("channel_name", channel.Name),
				attr.Int("channel_type", int(channel.Type)),
			)

			// 0 = GuildText, 2 = GuildVoice, 4 = GuildCategory, 5 = GuildNews, 10-12 = Threads
			if channel.Type == discordgo.ChannelTypeGuildCategory || channel.Type == discordgo.ChannelTypeGuildVoice {
				return CreateRoundOperationResult{
					Error: fmt.Errorf("invalid channel type for events: %d (Category/Voice)", channel.Type),
				}, nil
			}
		}

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
				// PRESERVED: old 3-field RSVP status grouping — may be reused in PWA
				// {Name: "✅ Accepted", Value: "-", Inline: true},
				// {Name: "❌ Declined", Value: "-", Inline: true},
				// {Name: "🤔 Tentative", Value: "-", Inline: true},
				{
					Name:   "👥 Participants",
					Value:  "-",
					Inline: false,
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

func (crm *createRoundManager) SendRoundEventURL(guildID string, channelID string, eventID string) (CreateRoundOperationResult, error) {
	return crm.operationWrapper(context.Background(), "SendRoundEventURL", func(ctx context.Context) (CreateRoundOperationResult, error) {
		eventURL := fmt.Sprintf("https://discord.com/events/%s/%s", guildID, eventID)

		msg, err := crm.session.ChannelMessageSend(channelID, eventURL)
		if err != nil {
			return CreateRoundOperationResult{Error: fmt.Errorf("failed to send event url message: %w", err)}, nil
		}

		return CreateRoundOperationResult{Success: msg}, nil
	})
}
