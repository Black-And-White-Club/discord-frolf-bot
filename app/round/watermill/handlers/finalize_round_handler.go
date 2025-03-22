package roundhandlers

import (
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round finalized event", attr.CorrelationIDFromMsg(msg))

	// Unmarshal the payload
	var payload roundevents.RoundFinalizedEmbedUpdatePayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Make sure we have an event message ID
	if payload.EventMessageID == nil {
		h.Logger.Error(ctx, "Missing event message ID in payload", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("missing event message ID in round finalized payload")
	}

	channelID := payload.DiscordChannelID

	// Update the round embed to finalized format
	_, err := h.RoundDiscord.GetFinalizeRoundManager().FinalizeScorecardEmbed(ctx, string(*payload.EventMessageID), channelID, payload)
	if err != nil {
		h.Logger.Error(ctx, "Failed to finalize scorecard embed", attr.Error(err))
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round finalized event", attr.CorrelationIDFromMsg(msg))

	// Create trace event
	tracePayload := map[string]interface{}{
		"round_id":   payload.RoundID,
		"event_type": "round_finalized",
		"status":     "scorecard_finalized",
		"message_id": payload.EventMessageID,
	}

	traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create trace event", attr.Error(err))
		return []*message.Message{}, nil
	}

	return []*message.Message{traceMsg}, nil
}
