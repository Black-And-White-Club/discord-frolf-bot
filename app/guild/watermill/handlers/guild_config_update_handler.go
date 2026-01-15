package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// / HandleGuildConfigUpdated handles successful guild config updates
func (h *GuildHandlers) HandleGuildConfigUpdated(ctx context.Context, payload *guildevents.GuildConfigUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.InfoContext(ctx, "Guild config updated",
		attr.String("guild_id", guildID),
		attr.String("updated_fields", fmt.Sprintf("%v", payload.UpdatedFields)))

	convertedConfig := convertGuildConfigFromShared(&payload.Config)
	if h.GuildConfigResolver != nil && convertedConfig != nil {
		h.GuildConfigResolver.HandleGuildConfigReceived(ctx, guildID, convertedConfig)
	}

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

	if h.SignupManager != nil && convertedConfig != nil && convertedConfig.SignupChannelID != "" {
		h.SignupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
	}

	// 1. UI FEEDBACK: Notify the admin that the update was successful
	if h.InteractionStore != nil && h.Session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.InteractionStore, guildID); err == nil {
			h.InteractionStore.Delete(ctx, guildID)

			successContent := "✅ **Configuration Updated Successfully!**"
			_, err = h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &successContent,
				Components: &[]discordgo.MessageComponent{}, // Remove any action buttons
			})
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send update success response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

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
		// Future: Trigger re-registration if command permissions are role-bound
	}

	return []handlerwrapper.Result{}, nil
}

// HandleGuildConfigUpdateFailed handles failed guild config update failures
func (h *GuildHandlers) HandleGuildConfigUpdateFailed(ctx context.Context, payload *guildevents.GuildConfigUpdateFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.ErrorContext(ctx, "Guild config update failed",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	// 2. UI FEEDBACK: Notify the admin of the failure
	if h.InteractionStore != nil && h.Session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.InteractionStore, guildID); err == nil {
			h.InteractionStore.Delete(ctx, guildID)

			failContent := fmt.Sprintf("❌ **Update Failed**\n\n**Reason:** %s", payload.Reason)
			_, err = h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &failContent,
				Components: &[]discordgo.MessageComponent{},
			})
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send update failure response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

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

	var convertedConfig *storage.GuildConfig
	if payload.Config.GuildID != "" || payload.Config.SignupChannelID != "" || payload.Config.LeaderboardChannelID != "" {
		convertedConfig = convertGuildConfigFromShared(&payload.Config)
	} else {
		convertedConfig = &storage.GuildConfig{GuildID: guildID, IsPlaceholder: true}
	}

	// 3. CACHE SYNC: Notify the resolver to populate the generic GuildConfigCache
	if h.GuildConfigResolver != nil && convertedConfig != nil {
		h.GuildConfigResolver.HandleGuildConfigReceived(ctx, guildID, convertedConfig)
	}

	// (Legacy support remains below)
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

	if h.SignupManager != nil && convertedConfig != nil && convertedConfig.SignupChannelID != "" {
		h.SignupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
	}

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
