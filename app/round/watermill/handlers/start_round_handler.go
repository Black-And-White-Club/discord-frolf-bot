package roundhandlers

import (
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundStarted handles the DiscordRoundStarted event and updates the Discord embed
func (h *RoundHandlers) HandleRoundStarted(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round started event", attr.CorrelationIDFromMsg(msg))

	// Unmarshal the payload
	var payload roundevents.DiscordRoundStartPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Make sure we have an event message ID
	if payload.EventMessageID == "" {
		h.Logger.Error(ctx, "Missing event message ID in payload", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("missing event message ID in round start payload")
	}

	channelID := payload.DiscordChannelID

	// Update the round embed to scorecard format
	if err := h.RoundDiscord.GetStartRoundManager().UpdateRoundToScorecard(ctx, channelID, string(payload.EventMessageID), &payload); err != nil {
		h.Logger.Error(ctx, "Failed to update round to scorecard", attr.Error(err))
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round start event", attr.CorrelationIDFromMsg(msg))

	// Create trace event inline without a separate function
	tracePayload := map[string]interface{}{
		"round_id":   payload.RoundID,
		"event_type": "round_started",
		"status":     "scorecard_updated",
		"message_id": payload.EventMessageID,
	}

	traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create trace event", attr.Error(err))
		return []*message.Message{}, nil
	}

	return []*message.Message{traceMsg}, nil
}
