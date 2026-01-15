package handlers

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleGuildConfigCreated handles successful guild config creation by registering all commands
func (h *GuildHandlers) HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.Logger.InfoContext(ctx, "Guild config created successfully - registering all commands for this guild",
		attr.String("guild_id", guildID))

	// 1. UI FEEDBACK: Notify the admin that setup is complete
	if h.InteractionStore != nil && h.Session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.InteractionStore, guildID); err == nil {
			// Clean up the store immediately
			h.InteractionStore.Delete(ctx, guildID)

			successContent := "✅ **Setup Complete!**\nAll server commands have been registered and are ready to use."
			_, err = h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &successContent,
				Components: &[]discordgo.MessageComponent{}, // Clear any setup buttons/modals
			})
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send setup success response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

	// 2. Register all bot commands for the successfully configured guild
	if err := h.GuildDiscord.RegisterAllCommands(guildID); err != nil {
		h.Logger.ErrorContext(ctx, "Failed to register all commands for guild after config creation",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return nil, fmt.Errorf("failed to register commands for guild %s: %w", guildID, err)
	}

	// 3. Make guild config available to runtime immediately
	convertedConfig := convertGuildConfigFromShared(&payload.Config)
	if convertedConfig != nil {
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

		// Track channels for reaction handling
		if h.SignupManager != nil && convertedConfig.SignupChannelID != "" {
			h.SignupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
		}
	}

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

	// 1. UI FEEDBACK: Notify the admin of the failure
	if h.InteractionStore != nil && h.Session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.InteractionStore, guildID); err == nil {
			h.InteractionStore.Delete(ctx, guildID)

			failContent := fmt.Sprintf("❌ **Setup Failed**\n\n**Reason:** %s\n\nPlease try running `/frolf-setup` again.", payload.Reason)
			_, err = h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &failContent,
				Components: &[]discordgo.MessageComponent{},
			})
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send setup failure response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

	return []handlerwrapper.Result{}, nil
}
