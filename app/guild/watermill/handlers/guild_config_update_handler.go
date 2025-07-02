package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGuildConfigUpdated handles guild config updates - may need to refresh command permissions
func (h *GuildHandlers) HandleGuildConfigUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigUpdated",
		&guildevents.GuildConfigUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigUpdatedPayload)
			guildID := string(p.GuildID)

			h.Logger.InfoContext(ctx, "Guild config updated",
				attr.String("guild_id", guildID),
				attr.String("updated_fields", fmt.Sprintf("%v", p.UpdatedFields)))

			// Check if role-related fields were updated that might affect command permissions
			needsCommandRefresh := false
			for _, field := range p.UpdatedFields {
				if field == "admin_role_id" || field == "editor_role_id" || field == "user_role_id" {
					needsCommandRefresh = true
					break
				}
			}

			if needsCommandRefresh {
				h.Logger.InfoContext(ctx, "Role configuration updated, refreshing command permissions",
					attr.String("guild_id", guildID))

				// For now, we just log this. In the future, we might need to:
				// 1. Update command permissions in Discord
				// 2. Re-register commands with new permission settings
				// For now, commands are already registered so no action needed
			}

			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigUpdateFailed handles guild config update failures
func (h *GuildHandlers) HandleGuildConfigUpdateFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigUpdateFailed",
		&guildevents.GuildConfigUpdateFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigUpdateFailedPayload)
			guildID := string(p.GuildID)

			h.Logger.ErrorContext(ctx, "Guild config update failed",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			// No action needed on Discord side for update failures - they're just notifications
			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigRetrieved handles successful guild config retrieval
func (h *GuildHandlers) HandleGuildConfigRetrieved(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigRetrieved",
		&guildevents.GuildConfigRetrievedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigRetrievedPayload)
			guildID := string(p.GuildID)

			h.Logger.InfoContext(ctx, "Guild config retrieved successfully",
				attr.String("guild_id", guildID))

			// No Discord action needed for config retrieval responses - they're informational
			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigRetrievalFailed handles failed guild config retrieval
func (h *GuildHandlers) HandleGuildConfigRetrievalFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigRetrievalFailed",
		&guildevents.GuildConfigRetrievalFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigRetrievalFailedPayload)
			guildID := string(p.GuildID)

			h.Logger.ErrorContext(ctx, "Guild config retrieval failed",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			// No Discord action needed for retrieval failures - they're informational
			return nil, nil
		},
	)(msg)
}
