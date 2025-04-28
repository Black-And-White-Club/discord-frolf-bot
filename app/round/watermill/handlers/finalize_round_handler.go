package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundFinalized",
		&roundevents.RoundFinalizedEmbedUpdatePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.RoundFinalizedEmbedUpdatePayload)

			// Validate input
			if uuid.UUID(p.RoundID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing RoundID in payload", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing RoundID in finalized payload")
			}
			if p.EventMessageID == nil || uuid.UUID(*p.EventMessageID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing event message ID in payload", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing event message ID in round finalized payload")
			}

			// Finalize the round embed
			_, err := h.RoundDiscord.GetFinalizeRoundManager().FinalizeScorecardEmbed(
				ctx,
				*p.EventMessageID,
				p.DiscordChannelID,
				*p,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to finalize scorecard embed", attr.Error(err))
				return nil, fmt.Errorf("failed to finalize scorecard embed: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully finalized round scorecard", attr.RoundID("round_id", p.RoundID))

			// Create trace event
			tracePayload := map[string]interface{}{
				"round_id":   p.RoundID,
				"event_type": "round_finalized",
				"status":     "scorecard_finalized",
				"message_id": p.EventMessageID,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event", attr.Error(err))
				return []*message.Message{}, nil
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
