package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGuildConfigCreated handles successful guild config creation by registering all commands
func (h *GuildHandlers) HandleGuildConfigCreated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigCreated",
		&guildevents.GuildConfigCreatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigCreatedPayload)
			guildID := string(p.GuildID)

			h.Logger.InfoContext(ctx, "Guild config created successfully - registering all commands",
				attr.String("guild_id", guildID))

			// Register all bot commands for the successfully configured guild
			if err := h.GuildDiscord.RegisterAllCommands(guildID); err != nil {
				h.Logger.ErrorContext(ctx, "Failed to register all commands for guild after config creation",
					attr.String("guild_id", guildID),
					attr.Error(err))
				return nil, fmt.Errorf("failed to register commands for guild %s: %w", guildID, err)
			}

			h.Logger.InfoContext(ctx, "Successfully registered all commands for guild after config creation",
				attr.String("guild_id", guildID))

			return nil, nil
		},
	)(msg)
}

// HandleGuildConfigCreationFailed handles failed guild config creation
func (h *GuildHandlers) HandleGuildConfigCreationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigCreationFailed",
		&guildevents.GuildConfigCreationFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigCreationFailedPayload)
			guildID := string(p.GuildID)

			h.Logger.WarnContext(ctx, "Guild config creation failed, commands remain limited to setup only",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			// Guild remains in setup-only mode - no additional command registration needed
			// Only /frolf-setup will be available until setup succeeds

			return nil, nil
		},
	)(msg)
}
