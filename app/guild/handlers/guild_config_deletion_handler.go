package handlers

import (
	"context"
	"fmt"
	"maps"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleGuildConfigDeleted handles guild config deletion by unregistering commands
// and best-effort cleanup of Discord resources. This handler must be safe for
// retries and replays (JetStream at-least-once).
func (h *GuildHandlers) HandleGuildConfigDeleted(ctx context.Context, payload *guildevents.GuildConfigDeletedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.logger.InfoContext(ctx, "Guild config deleted - starting cleanup",
		attr.String("guild_id", guildID))

	if h.guildConfigResolver != nil {
		h.guildConfigResolver.ClearInflightRequest(ctx, guildID)
	}

	// Unregister commands. Fail fast so the message can be retried if
	// command unregistration fails (this is an important cleanup step).
	if err := h.service.UnregisterAllCommands(guildID); err != nil {
		h.logger.ErrorContext(ctx, "Failed to unregister all commands",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return nil, fmt.Errorf("failed to unregister all commands: %w", err)
	}

	// Log success for test expectations and observability
	h.logger.InfoContext(ctx, "Successfully unregistered all commands",
		attr.String("guild_id", guildID))

	results := make(map[string]guildtypes.DeletionResult)

	// Delegate deletion to the ResetManager (canonical owner of deletion logic).
	rs := payload.ResourceState
	if !rs.IsEmpty() {
		var err error
		if h.service != nil && h.service.GetResetManager() != nil {
			results, err = h.service.GetResetManager().DeleteResources(ctx, guildID, rs)
			if err != nil {
				h.logger.ErrorContext(ctx, "ResetManager.DeleteResources returned error",
					attr.String("guild_id", guildID),
					attr.Error(err))
			}
		} else {
			h.logger.WarnContext(ctx, "No reset manager available to delete resources",
				attr.String("guild_id", guildID))
		}
	}

	// Follow-up interaction (best-effort UX)
	h.sendDeletionSummary(ctx, guildID, results)

	// Publish deletion results event (best-effort)
	if len(results) == 0 {
		return []handlerwrapper.Result{}, nil
	}

	if payload.ResourceState.Results == nil {
		payload.ResourceState.Results = make(map[string]guildtypes.DeletionResult)
	}
	maps.Copy(payload.ResourceState.Results, results)

	out := guildevents.GuildConfigDeletionResultsPayloadV1{
		GuildID:       payload.GuildID,
		ResourceState: payload.ResourceState,
		Results:       payload.ResourceState.Results,
	}

	return []handlerwrapper.Result{
		{
			Topic:   guildevents.GuildConfigDeletionResultsV1,
			Payload: &out,
		},
	}, nil
}

// HandleGuildConfigDeletionFailed handles failed guild config deletion
func (h *GuildHandlers) HandleGuildConfigDeletionFailed(ctx context.Context, payload *guildevents.GuildConfigDeletionFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	guildID := string(payload.GuildID)

	h.logger.WarnContext(ctx, "Guild config deletion failed",
		attr.String("guild_id", guildID),
		attr.String("reason", payload.Reason))

	if h.interactionStore == nil || h.session == nil {
		return []handlerwrapper.Result{}, nil
	}

	// UPDATED: Use the bridge utility with context
	if interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, guildID); err == nil {
		// Clean up immediately
		h.interactionStore.Delete(ctx, guildID)

		content := fmt.Sprintf(
			"❌ Failed to reset server configuration.\n\n**Reason:** %s\n\nPlease try again.",
			payload.Reason,
		)

		_, err := h.session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
			Content:    &content,
			Components: &[]discordgo.MessageComponent{},
		})
		if err != nil {
			h.logger.ErrorContext(ctx, "Failed to send deletion failure response",
				attr.String("guild_id", guildID),
				attr.Error(err))
		}
	}

	return []handlerwrapper.Result{}, nil
}

/*
	Helpers
*/

func (h *GuildHandlers) sendDeletionSummary(
	ctx context.Context,
	guildID string,
	results map[string]guildtypes.DeletionResult,
) {
	if h.interactionStore == nil || h.session == nil {
		return
	}

	// UPDATED: Use the bridge utility to replace manual Get + Assertion
	interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, guildID)
	if err != nil {
		// If it's not in the store, we can't send a summary, just exit
		return
	}

	// Clean up the cache now that we've retrieved it
	h.interactionStore.Delete(ctx, guildID)

	summary := "✅ Server configuration reset completed.\n\n"
	summary += "Bot commands have been unregistered. Run `/frolf-setup` when you're ready.\n\n"

	if len(results) > 0 {
		summary += "Deletion results:\n"
		for k, r := range results {
			if r.Status == "success" {
				summary += fmt.Sprintf("- %s: ✅\n", k)
			} else {
				summary += fmt.Sprintf("- %s: ❌ %s\n", k, r.Error)
			}
		}
	}

	_, err = h.session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content:    &summary,
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to send deletion summary",
			attr.String("guild_id", guildID),
			attr.Error(err))
	}
}
