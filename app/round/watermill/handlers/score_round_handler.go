package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleParticipantScoreUpdated handles a successful participant score update event.
func (h *RoundHandlers) HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleParticipantScoreUpdated",
		&roundevents.ParticipantScoreUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatePayload, ok := payload.(*roundevents.ParticipantScoreUpdatedPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleParticipantScoreUpdated")
			}

			scorePtr := &updatePayload.Score

			// Capture both result and error
			result, err := h.RoundDiscord.GetScoreRoundManager().SendScoreUpdateConfirmation(
				ctx,
				updatePayload.ChannelID,
				updatePayload.Participant,
				scorePtr,
			)
			// Check both possible error sources
			if err != nil {
				return nil, fmt.Errorf("failed to send score update confirmation: %w", err)
			}
			if result.Error != nil {
				return nil, fmt.Errorf("error in send score update confirmation: %w", result.Error)
			}

			tracePayload := map[string]interface{}{
				"round_id":    updatePayload.RoundID,
				"event_type":  "participant_score_updated",
				"status":      "confirmation_sent",
				"participant": updatePayload.Participant,
				"score":       updatePayload.Score,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}

// HandleScoreUpdateError processes a failed score update event.
func (h *RoundHandlers) HandleScoreUpdateError(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScoreUpdateError",
		&roundevents.RoundScoreUpdateErrorPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			errorPayload, ok := payload.(*roundevents.RoundScoreUpdateErrorPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleScoreUpdateError")
			}

			if errorPayload.Error == "" {
				return nil, fmt.Errorf("received empty error message in HandleScoreUpdateError")
			}

			_, err := h.RoundDiscord.GetScoreRoundManager().SendScoreUpdateError(ctx, errorPayload.ScoreUpdateRequest.Participant, errorPayload.Error)
			if err != nil {
				return nil, fmt.Errorf("failed to send score update error notification: %w", err)
			}

			tracePayload := map[string]interface{}{
				"round_id":    errorPayload.ScoreUpdateRequest.RoundID,
				"event_type":  "score_update_error",
				"status":      "error_notification_sent",
				"participant": errorPayload.ScoreUpdateRequest.Participant,
				"error":       errorPayload.Error,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
