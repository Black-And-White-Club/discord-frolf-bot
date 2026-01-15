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
	if channelID == "" && payload.Config != nil {
		channelID = payload.Config.EventChannelID
	}
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

	// We don't return a trace event here to avoid downstream publish
	// failures causing the handler (and its side-effects) to be retried.
	// Returning an empty result set will ensure the original message is
	// Acked after successful handling.
	return []handlerwrapper.Result{}, nil
}
