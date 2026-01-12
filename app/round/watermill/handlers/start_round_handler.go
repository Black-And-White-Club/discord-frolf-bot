package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundStarted updates the Discord embed when a round starts.
func (h *RoundHandlers) HandleRoundStarted(ctx context.Context, payload *roundevents.DiscordRoundStartPayloadV1) ([]handlerwrapper.Result, error) {
	if payload.EventMessageID == "" {
		return nil, fmt.Errorf("missing event message ID in round start payload")
	}

	// Use the channel ID from the payload instead of config
	channelID := payload.DiscordChannelID
	if channelID == "" {
		// Fallback to config if payload channel ID is empty
		channelID = h.Config.GetEventChannelID()
	}

	eventMessageID := payload.EventMessageID

	// Update round to scorecard
	_, err := h.RoundDiscord.GetStartRoundManager().UpdateRoundToScorecard(ctx, channelID, eventMessageID, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to update round to scorecard: %w", err)
	}

	tracePayload := map[string]interface{}{
		"guild_id":   payload.GuildID,
		"round_id":   payload.RoundID,
		"event_type": "round_started",
		"status":     "scorecard_updated",
		"message_id": eventMessageID,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundTraceEventV1,
			Payload: tracePayload,
		},
	}, nil
}
