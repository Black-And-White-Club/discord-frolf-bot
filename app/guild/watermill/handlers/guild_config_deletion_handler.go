package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// HandleGuildConfigDeleted handles guild config deletion by unregistering all commands
func (h *GuildHandlers) HandleGuildConfigDeleted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeleted",
		&guildevents.GuildConfigDeletedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletedPayloadV1)
			guildID := string(p.GuildID)

			h.Logger.InfoContext(ctx, "Guild config deleted - unregistering all commands",
				attr.String("guild_id", guildID))

			// Unregister all bot commands for the guild
			if err := h.GuildDiscord.UnregisterAllCommands(guildID); err != nil {
				h.Logger.ErrorContext(ctx, "Failed to unregister all commands for guild after config deletion",
					attr.String("guild_id", guildID),
					attr.Error(err))
				return nil, fmt.Errorf("failed to unregister commands for guild %s: %w", guildID, err)
			}

			h.Logger.InfoContext(ctx, "Successfully unregistered all commands for guild after config deletion",
				attr.String("guild_id", guildID))

			// Send user feedback if we have a stored interaction
			if h.InteractionStore != nil && h.Session != nil {
				if interactionData, exists := h.InteractionStore.Get(guildID); exists {
					h.InteractionStore.Delete(guildID)
					if interaction, ok := interactionData.(*discordgo.Interaction); ok {
						content := "✅ Server configuration reset successfully!\n\n" +
							"Bot commands have been unregistered. Run `/frolf-setup` when you're ready to set up again."
						_, err := h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
							Content:    &content,
							Components: &[]discordgo.MessageComponent{},
						})
						if err != nil {
							h.Logger.ErrorContext(ctx, "Failed to send success followup to user",
								attr.String("guild_id", guildID),
								attr.Error(err))
							// Don't return error - the operation succeeded even if followup failed
						}
					}
				}
			}

			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigDeletionFailed handles failed guild config deletion
func (h *GuildHandlers) HandleGuildConfigDeletionFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeletionFailed",
		&guildevents.GuildConfigDeletionFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletionFailedPayloadV1)
			guildID := string(p.GuildID)

			h.Logger.WarnContext(ctx, "Guild config deletion failed, commands remain active",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			// Send user feedback if we have a stored interaction
			if h.InteractionStore != nil && h.Session != nil {
				if interactionData, exists := h.InteractionStore.Get(guildID); exists {
					h.InteractionStore.Delete(guildID)
					if interaction, ok := interactionData.(*discordgo.Interaction); ok {
						errorMsg := fmt.Sprintf("❌ Failed to reset server configuration.\n\n**Reason:** %s\n\n"+
							"Please try again or contact support if the issue persists.", p.Reason)
						_, err := h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
							Content:    &errorMsg,
							Components: &[]discordgo.MessageComponent{},
						})
						if err != nil {
							h.Logger.ErrorContext(ctx, "Failed to send failure followup to user",
								attr.String("guild_id", guildID),
								attr.Error(err))
						}
					}
				}
			}

			return nil, nil
		},
	)(msg)
}
