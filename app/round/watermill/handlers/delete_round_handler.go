package roundhandlers

import (
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundDeleted handles the RoundDeleted event by removing the corresponding Discord message.
func (h *RoundHandlers) HandleRoundDeleted(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round deleted event", attr.CorrelationIDFromMsg(msg))

	// Unmarshal the payload
	var payload roundevents.RoundDeletedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Validate that RoundID is provided
	if payload.RoundID == 0 {
		h.Logger.Error(ctx, "Missing RoundID in payload", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("missing RoundID in round deleted payload")
	}

	// Validate that EventMessageID is provided
	if payload.EventMessageID == "" {
		h.Logger.Error(ctx, "Missing EventMessageID in payload", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("event message ID is required but missing for round deleted event")
	}

	// Attempt to delete the Discord embed message
	success, err := h.RoundDiscord.GetDeleteRoundManager().DeleteEmbed(ctx, payload.EventMessageID, h.Config.Discord.ChannelID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to delete round embed message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to delete round embed message: %w", err)
	}

	// Log whether the deletion was successful
	if !success {
		h.Logger.Warn(ctx, "Round embed message was not deleted successfully", attr.CorrelationIDFromMsg(msg))
	} else {
		h.Logger.Info(ctx, "Successfully deleted round embed message", attr.CorrelationIDFromMsg(msg))
	}

	// Create a trace event
	tracePayload := map[string]interface{}{
		"round_id":   payload.RoundID,
		"event_type": "round_deleted",
		"status":     "embed_deleted",
		"message_id": payload.EventMessageID,
	}

	traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create trace event", attr.Error(err))
		return []*message.Message{}, nil
	}

	return []*message.Message{traceMsg}, nil
}
