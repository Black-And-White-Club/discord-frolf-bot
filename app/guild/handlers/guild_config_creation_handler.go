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

	h.logger.InfoContext(ctx, "Guild config created successfully - registering all commands for this guild",
		attr.String("guild_id", guildID))

	// 1. UI FEEDBACK: Notify the admin that setup is complete
	if h.interactionStore != nil && h.session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, guildID); err == nil {
			// Clean up the store immediately
			h.interactionStore.Delete(ctx, guildID)

			successContent := "✅ **Setup Complete!**\nAll server commands have been registered and are ready to use."
			_, err = h.session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &successContent,
				Components: &[]discordgo.MessageComponent{}, // Clear any setup buttons/modals
			})
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to send setup success response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

	// 2. Register all bot commands for the successfully configured guild
	if err := h.service.RegisterAllCommands(guildID); err != nil {
		h.logger.ErrorContext(ctx, "Failed to register all commands for guild after config creation",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return nil, fmt.Errorf("failed to register commands for guild %s: %w", guildID, err)
	}

	// 3. Make guild config available to runtime immediately
	convertedConfig := convertGuildConfigFromShared(&payload.Config)
	if convertedConfig != nil {

		// Notify resolver to unblock any pending config lookups
		if h.guildConfigResolver != nil {
			h.guildConfigResolver.HandleGuildConfigReceived(ctx, guildID, convertedConfig)
		}

		// Track channels for reaction handling
		if h.signupManager != nil && convertedConfig.SignupChannelID != "" {
			h.signupManager.TrackChannelForReactions(convertedConfig.SignupChannelID)
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

	h.logger.WarnContext(ctx, "Guild config creation failed, commands remain limited to setup only",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	// 1. UI FEEDBACK: Notify the admin of the failure
	if h.interactionStore != nil && h.session != nil {
		if interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, guildID); err == nil {
			h.interactionStore.Delete(ctx, guildID)

			failContent := fmt.Sprintf("❌ **Setup Failed**\n\n**Reason:** %s\n\nPlease try running `/frolf-setup` again.", payload.Reason)
			_, err = h.session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
				Content:    &failContent,
				Components: &[]discordgo.MessageComponent{},
			})
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to send setup failure response",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		}
	}

	return []handlerwrapper.Result{}, nil
}
