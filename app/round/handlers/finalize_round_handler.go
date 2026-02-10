package handlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	discordChannelID := h.config.GetEventChannelID()

	// Get message ID from context (set by wrapper from message metadata)
	discordMessageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || discordMessageID == "" {
		return nil, fmt.Errorf("missing discord_message_id in metadata for round finalized")
	}

	// Convert the Discord-specific payload into the embed update payload
	// expected by the FinalizeScorecardEmbed manager.
	embedPayload := roundevents.RoundFinalizedEmbedUpdatePayloadV1{
		GuildID:          payload.GuildID,
		RoundID:          payload.RoundID,
		Title:            payload.Title,
		StartTime:        payload.StartTime,
		Location:         payload.Location,
		Participants:     payload.Participants,
		Teams:            payload.Teams,
		EventMessageID:   payload.EventMessageID,
		DiscordChannelID: payload.DiscordChannelID,
	}

	// Cache the payload for potential points updates
	cacheKey := fmt.Sprintf("round_payload:%s", payload.RoundID)
	err := h.interactionStore.Set(ctx, cacheKey, embedPayload)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to cache round payload", "round_id", payload.RoundID, "error", err)
	}

	// Get the FinalizeRoundManager and finalize the round embed
	finalizeRoundManager := h.service.GetFinalizeRoundManager()
	finalizeResult, err := finalizeRoundManager.FinalizeScorecardEmbed(
		ctx,
		discordMessageID, // Pass message ID obtained from context
		discordChannelID, // Pass channel ID from config
		embedPayload,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize scorecard embed: %w", err)
	}

	if finalizeResult.Error != nil {
		return nil, fmt.Errorf("finalize scorecard embed operation failed: %w", finalizeResult.Error)
	}

	// Set the native Discord Scheduled Event to COMPLETED (best-effort).
	if payload.DiscordEventID != "" {
		session := h.service.GetSession()
		_, err = session.GuildScheduledEventEdit(string(payload.GuildID), payload.DiscordEventID, &discordgo.GuildScheduledEventParams{
			Status: discordgo.GuildScheduledEventStatusCompleted,
		})
		if err != nil {
			h.logger.WarnContext(ctx, "failed to set native event to COMPLETED",
				"discord_event_id", payload.DiscordEventID,
				"error", err,
			)
		}
	}

	// We intentionally do not emit a trace event here. Returning result
	// messages causes Watermill to attempt publishing; if the trace topic
	// has no configured consumer/stream, publish will fail and the input
	// message will be Nacked, retrying the handler and duplicating side
	// effects. Returning an empty result set avoids that.
	return []handlerwrapper.Result{}, nil
}
