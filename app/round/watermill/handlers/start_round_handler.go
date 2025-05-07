package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundStarted updates the Discord embed when a round starts.
func (h *RoundHandlers) HandleRoundStarted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundStarted",
		&roundevents.DiscordRoundStartPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			startPayload, ok := payload.(*roundevents.DiscordRoundStartPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleRoundStarted")
			}

			if startPayload.EventMessageID == "" {
				return nil, fmt.Errorf("missing event message ID in round start payload")
			}

			// Convert EventMessageID to string
			eventMessageID := startPayload.EventMessageID

			// Capture both return values from UpdateRoundToScorecard
			_, err := h.RoundDiscord.GetStartRoundManager().UpdateRoundToScorecard(ctx, h.Config.Discord.ChannelID, eventMessageID, startPayload)
			if err != nil {
				return nil, fmt.Errorf("failed to update round to scorecard: %w", err)
			}

			tracePayload := map[string]interface{}{
				"round_id":   startPayload.RoundID,
				"event_type": "round_started",
				"status":     "scorecard_updated",
				"message_id": eventMessageID,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
