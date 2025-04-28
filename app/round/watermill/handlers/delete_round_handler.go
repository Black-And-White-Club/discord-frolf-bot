package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleRoundDeleted handles the RoundDeleted event using the standardized wrapper
func (h *RoundHandlers) HandleRoundDeleted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundDeleted",
		&roundevents.RoundDeletedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.RoundDeletedPayload)

			// Validate input
			if uuid.UUID(p.RoundID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing RoundID in payload", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing RoundID in round deleted payload")
			}
			if uuid.UUID(p.EventMessageID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing EventMessageID in payload", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("event message ID is required but missing for round deleted event")
			}

			// Attempt deletion
			result, err := h.RoundDiscord.GetDeleteRoundManager().DeleteRoundEventEmbed(ctx, p.EventMessageID, h.Config.Discord.ChannelID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to delete round embed message", attr.Error(err))
				return nil, fmt.Errorf("failed to delete round embed message: %w", err)
			}

			// Assert Success field to bool
			success, ok := result.Success.(bool)
			if !ok {
				h.Logger.ErrorContext(ctx, "Unexpected type for result.Success", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("unexpected type for result.Success in DeleteRoundEventEmbed")
			}

			if !success {
				h.Logger.WarnContext(ctx, "Round embed message was not deleted successfully", attr.RoundID("round_id", p.RoundID))
			} else {
				h.Logger.InfoContext(ctx, "Successfully deleted round embed message", attr.RoundID("round_id", p.RoundID))
			}

			// Create trace message
			tracePayload := map[string]interface{}{
				"round_id":   p.RoundID,
				"event_type": "round_deleted",
				"status":     "embed_deleted",
				"message_id": p.EventMessageID,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event", attr.Error(err))
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
