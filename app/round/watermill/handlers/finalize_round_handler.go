package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	discordChannelID := h.Config.GetEventChannelID()

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
		EventMessageID:   payload.EventMessageID,
		DiscordChannelID: payload.DiscordChannelID,
	}

	// Get the FinalizeRoundManager and finalize the round embed
	finalizeRoundManager := h.RoundDiscord.GetFinalizeRoundManager()
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

	// Create trace event
	tracePayload := map[string]interface{}{
		"guild_id":           payload.GuildID,
		"round_id":           payload.RoundID,
		"event_type":         "round_finalized",
		"status":             "scorecard_finalized_display",
		"discord_message_id": discordMessageID,
		"channel_id":         discordChannelID,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundTraceEventV1,
			Payload: tracePayload,
		},
	}, nil
}
