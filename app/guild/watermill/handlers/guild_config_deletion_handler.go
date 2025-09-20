package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGuildConfigDeleted handles guild config deletion by unregistering all commands
func (h *GuildHandlers) HandleGuildConfigDeleted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeleted",
		&guildevents.GuildConfigDeletedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletedPayload)
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

			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigDeletionFailed handles failed guild config deletion
func (h *GuildHandlers) HandleGuildConfigDeletionFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeletionFailed",
		&guildevents.GuildConfigDeletionFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletionFailedPayload)
			guildID := string(p.GuildID)

			h.Logger.WarnContext(ctx, "Guild config deletion failed, commands remain active",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			// Guild config deletion failed, so commands should remain active
			// No action needed

			return nil, nil
		},
	)(msg)
}
