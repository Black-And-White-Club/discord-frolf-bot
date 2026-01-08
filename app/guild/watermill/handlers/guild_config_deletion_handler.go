package handlers

import (
	"context"
	"errors"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

/*
Result keys – centralized to avoid string drift across publishers/consumers
*/
const (
	resultSignupMessage      = "signup_message"
	resultSignupChannel      = "signup_channel"
	resultEventChannel       = "event_channel"
	resultLeaderboardChannel = "leaderboard_channel"
	resultUserRole           = "user_role"
	resultEditorRole         = "editor_role"
	resultAdminRole          = "admin_role"
)

// HandleGuildConfigDeleted handles guild config deletion by unregistering commands
// and best-effort cleanup of Discord resources. This handler must be safe for
// retries and replays (JetStream at-least-once).
func (h *GuildHandlers) HandleGuildConfigDeleted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeleted",
		&guildevents.GuildConfigDeletedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletedPayloadV1)
			guildID := string(p.GuildID)

			h.Logger.InfoContext(ctx, "Guild config deleted - starting cleanup",
				attr.String("guild_id", guildID))

			if h.GuildConfigResolver != nil {
				h.GuildConfigResolver.ClearInflightRequest(ctx, guildID)
			}

			// Unregister commands. Fail fast so the message can be retried if
			// command unregistration fails (this is an important cleanup step).
			if err := h.GuildDiscord.UnregisterAllCommands(guildID); err != nil {
				h.Logger.ErrorContext(ctx, "Failed to unregister all commands",
					attr.String("guild_id", guildID),
					attr.Error(err))
				return nil, fmt.Errorf("failed to unregister all commands: %w", err)
			}

			// Log success for test expectations and observability
			h.Logger.InfoContext(ctx, "Successfully unregistered all commands",
				attr.String("guild_id", guildID))

			results := make(map[string]guildtypes.DeletionResult)

			// Delegate deletion to the ResetManager (canonical owner of deletion logic).
			rs := p.ResourceState
			if !rs.IsEmpty() {
				var err error
				if h.GuildDiscord != nil && h.GuildDiscord.GetResetManager() != nil {
					results, err = h.GuildDiscord.GetResetManager().DeleteResources(ctx, guildID, rs)
					if err != nil {
						h.Logger.ErrorContext(ctx, "ResetManager.DeleteResources returned error",
							attr.String("guild_id", guildID),
							attr.Error(err))
					}
				} else {
					h.Logger.WarnContext(ctx, "No reset manager available to delete resources",
						attr.String("guild_id", guildID))
				}
			}

			// Follow-up interaction (best-effort UX)
			h.sendDeletionSummary(ctx, guildID, results)

			// Publish deletion results event (best-effort)
			if len(results) == 0 {
				return nil, nil
			}

			if p.ResourceState.Results == nil {
				p.ResourceState.Results = make(map[string]guildtypes.DeletionResult)
			}
			for k, v := range results {
				p.ResourceState.Results[k] = v
			}

			out := guildevents.GuildConfigDeletionResultsPayloadV1{
				GuildID:       p.GuildID,
				ResourceState: p.ResourceState,
				Results:       p.ResourceState.Results,
			}

			resultMsg, err := h.Helpers.CreateNewMessage(
				out,
				guildevents.GuildConfigDeletionResultsV1,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create deletion results event",
					attr.String("guild_id", guildID),
					attr.Error(err))
				return nil, nil
			}

			resultMsg.Metadata.Set("guild_id", guildID)
			return []*message.Message{resultMsg}, nil
		},
	)(msg)
}

// HandleGuildConfigDeletionFailed handles failed guild config deletion
func (h *GuildHandlers) HandleGuildConfigDeletionFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildConfigDeletionFailed",
		&guildevents.GuildConfigDeletionFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*guildevents.GuildConfigDeletionFailedPayloadV1)
			guildID := string(p.GuildID)

			h.Logger.WarnContext(ctx, "Guild config deletion failed",
				attr.String("guild_id", guildID),
				attr.String("reason", p.Reason))

			if h.InteractionStore == nil || h.Session == nil {
				return nil, nil
			}

			if interactionData, ok := h.InteractionStore.Get(guildID); ok {
				h.InteractionStore.Delete(guildID)

				if interaction, ok := interactionData.(*discordgo.Interaction); ok {
					content := fmt.Sprintf(
						"❌ Failed to reset server configuration.\n\n**Reason:** %s\n\nPlease try again.",
						p.Reason,
					)

					_, err := h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
						Content:    &content,
						Components: &[]discordgo.MessageComponent{},
					})
					if err != nil {
						h.Logger.ErrorContext(ctx, "Failed to send deletion failure response",
							attr.String("guild_id", guildID),
							attr.Error(err))
					}
				}
			}

			return nil, nil
		},
	)(msg)
}

/*
	Helpers
*/

func (h *GuildHandlers) sendDeletionSummary(
	ctx context.Context,
	guildID string,
	results map[string]guildtypes.DeletionResult,
) {
	if h.InteractionStore == nil || h.Session == nil {
		return
	}

	interactionData, ok := h.InteractionStore.Get(guildID)
	if !ok {
		return
	}
	h.InteractionStore.Delete(guildID)

	interaction, ok := interactionData.(*discordgo.Interaction)
	if !ok {
		return
	}

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

	_, err := h.Session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content:    &summary,
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		h.Logger.ErrorContext(ctx, "Failed to send deletion summary",
			attr.String("guild_id", guildID),
			attr.Error(err))
	}
}

func isDiscordNotFound(err error) bool {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		switch restErr.Message.Code {
		case discordgo.ErrCodeUnknownChannel,
			discordgo.ErrCodeUnknownMessage,
			discordgo.ErrCodeUnknownRole:
			return true
		}
	}
	return false
}
