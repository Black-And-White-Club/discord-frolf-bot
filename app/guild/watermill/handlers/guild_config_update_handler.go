package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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

			// Convert the config and notify the resolver that we received the response
			var convertedConfig *storage.GuildConfig
			if p.Config.GuildID != "" || p.Config.SignupChannelID != "" || p.Config.LeaderboardChannelID != "" {
				convertedConfig = convertGuildConfigFromShared(&p.Config)
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
				attr.String("signup_channel_id", p.Config.SignupChannelID))

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

			// The resolver will handle timeouts and retries automatically
			// This is just an informational event

			return nil, nil
		},
	)(msg)
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
