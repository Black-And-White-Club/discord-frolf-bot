package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGuildConfigCreated handles successful guild config creation by registering all commands
func (h *GuildHandlers) HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.InfoContext(ctx, "Guild config created successfully - registering all commands for this guild",
		attr.String("guild_id", guildID))

	// Register all bot commands for the successfully configured guild
	if err := h.GuildDiscord.RegisterAllCommands(guildID); err != nil {
		h.Logger.ErrorContext(ctx, "Failed to register all commands for guild after config creation",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return nil, fmt.Errorf("failed to register commands for guild %s: %w", guildID, err)
	}

	// Make guild config available to runtime immediately (not just on retrieval events)
	convertedConfig := convertGuildConfigFromShared(&payload.Config)
	if convertedConfig != nil {
		// Persist into in-memory runtime config so other handlers (e.g., leaderboard embeds) can resolve channels
		if h.Config != nil {
			h.Config.UpdateGuildConfig(
				convertedConfig.GuildID,
				convertedConfig.SignupChannelID,
				convertedConfig.EventChannelID,
				convertedConfig.LeaderboardChannelID,
				convertedConfig.SignupMessageID,
				convertedConfig.RegisteredRoleID,
				convertedConfig.AdminRoleID,
				convertedConfig.RoleMappings,
			)
		}

		// Notify resolver to unblock any pending config lookups
		if h.GuildConfigResolver != nil {
			h.GuildConfigResolver.HandleGuildConfigReceived(ctx, guildID, convertedConfig)
		}

		// Track channels for reaction handling to avoid unnecessary backend requests
		if h.SignupManager != nil && convertedConfig.SignupChannelID != "" {
			h.SignupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
		}
	}

	h.Logger.InfoContext(ctx, "Successfully registered all commands and cached guild config",
		attr.String("guild_id", guildID))

	return []handlerwrapper.Result{}, nil
}

// HandleGuildConfigCreationFailed handles failed guild config creation
func (h *GuildHandlers) HandleGuildConfigCreationFailed(ctx context.Context, payload *guildevents.GuildConfigCreationFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.WarnContext(ctx, "Guild config creation failed, commands remain limited to setup only",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	// Guild remains in setup-only mode - no additional command registration needed
	// Only /frolf-setup will be available until setup succeeds

	return []handlerwrapper.Result{}, nil
}
