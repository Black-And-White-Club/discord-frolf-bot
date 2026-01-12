package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// HandleGuildConfigUpdated handles guild config updates - may need to refresh command permissions
func (h *GuildHandlers) HandleGuildConfigUpdated(ctx context.Context, payload *guildevents.GuildConfigUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.InfoContext(ctx, "Guild config updated",
		attr.String("guild_id", guildID),
		attr.String("updated_fields", fmt.Sprintf("%v", payload.UpdatedFields)))

	// Check if role-related fields were updated that might affect command permissions
	needsCommandRefresh := false
	for _, field := range payload.UpdatedFields {
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

	return []handlerwrapper.Result{}, nil
}

// HandleGuildConfigUpdateFailed handles guild config update failures
func (h *GuildHandlers) HandleGuildConfigUpdateFailed(ctx context.Context, payload *guildevents.GuildConfigUpdateFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.ErrorContext(ctx, "Guild config update failed",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	// No action needed on Discord side for update failures - they're just notifications
	return []handlerwrapper.Result{}, nil
}

// HandleGuildConfigRetrieved handles successful guild config retrieval
func (h *GuildHandlers) HandleGuildConfigRetrieved(ctx context.Context, payload *guildevents.GuildConfigRetrievedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.InfoContext(ctx, "Guild config retrieved successfully",
		attr.String("guild_id", guildID))

	// Convert the config and notify the resolver that we received the response
	var convertedConfig *storage.GuildConfig
	if payload.Config.GuildID != "" || payload.Config.SignupChannelID != "" || payload.Config.LeaderboardChannelID != "" {
		convertedConfig = convertGuildConfigFromShared(&payload.Config)
	} else {
		// Provide a minimal placeholder to avoid nil dereference in consumers/tests
		convertedConfig = &storage.GuildConfig{GuildID: guildID}
	}

	// Persist into in-memory runtime config (single-tenant / legacy handlers rely on this)
	if h.Config != nil && convertedConfig != nil {
		roleMappings := map[string]string{}
		for k, v := range convertedConfig.RoleMappings {
			roleMappings[k] = v
		}
		h.Config.UpdateGuildConfig(
			convertedConfig.GuildID,
			convertedConfig.SignupChannelID,
			convertedConfig.EventChannelID,
			convertedConfig.LeaderboardChannelID,
			convertedConfig.SignupMessageID,
			convertedConfig.RegisteredRoleID,
			convertedConfig.AdminRoleID,
			roleMappings,
		)
		h.Logger.InfoContext(ctx, "In-memory guild config updated",
			attr.String("guild_id", guildID),
			attr.String("event_channel_id", convertedConfig.EventChannelID),
			attr.String("signup_channel_id", convertedConfig.SignupChannelID),
		)
	}

	// Persist into in-memory config so other handlers (e.g., round joins) can resolve channel IDs
	if h.Config != nil && convertedConfig != nil {
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
	if h.GuildConfigResolver != nil {
		h.GuildConfigResolver.HandleGuildConfigReceived(ctx, guildID, convertedConfig)
	}

	// Track channels for reaction handling to avoid unnecessary backend requests
	if h.SignupManager != nil && convertedConfig != nil {
		if convertedConfig.SignupChannelID != "" {
			h.SignupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
		}
	}

	h.Logger.InfoContext(ctx, "Guild config retrieved and available from backend",
		attr.String("guild_id", guildID),
		attr.String("signup_channel_id", payload.Config.SignupChannelID))

	return []handlerwrapper.Result{}, nil
}

// HandleGuildConfigRetrievalFailed handles failed guild config retrieval
func (h *GuildHandlers) HandleGuildConfigRetrievalFailed(ctx context.Context, payload *guildevents.GuildConfigRetrievalFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.ErrorContext(ctx, "Guild config retrieval failed",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	// The resolver will handle timeouts and retries automatically
	// This is just an informational event

	return []handlerwrapper.Result{}, nil
}

// convertGuildConfigFromShared converts a guildtypes.GuildConfig to a storage.GuildConfig
func convertGuildConfigFromShared(src *guildtypes.GuildConfig) *storage.GuildConfig {
	if src == nil {
		return nil
	}
	// Always use sharedtypes.UserRoleEnum constants for mapping keys
	roleMappings := map[string]string{
		string(sharedtypes.UserRoleUser):   src.UserRoleID,
		string(sharedtypes.UserRoleEditor): src.EditorRoleID,
		string(sharedtypes.UserRoleAdmin):  src.AdminRoleID,
	}
	return &storage.GuildConfig{
		GuildID:              string(src.GuildID),
		SignupChannelID:      src.SignupChannelID,
		SignupMessageID:      src.SignupMessageID,
		EventChannelID:       src.EventChannelID,
		LeaderboardChannelID: src.LeaderboardChannelID,
		RegisteredRoleID:     src.UserRoleID, // Map UserRoleID to RegisteredRoleID
		AdminRoleID:          src.AdminRoleID,
		RoleMappings:         roleMappings,
		SignupEmoji:          src.SignupEmoji,
		CachedAt:             time.Now(),
		RefreshedAt:          time.Now(),
		IsPlaceholder:        false,
		IsRequestPending:     false,
	}
}
