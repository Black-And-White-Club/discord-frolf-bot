package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleParticipantScoreUpdated handles a successful participant score update event from the backend.
// It now calls the scoreround.UpdateScoreEmbed function to update the scorecard.
// The incoming payload no longer needs the full participant list.
func (h *RoundHandlers) HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper( // Assuming handlerWrapper is defined elsewhere
		"HandleParticipantScoreUpdated",
		// Expecting a simpler payload without the full participant list
		&roundevents.ParticipantScoreUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) { // Corrected return type here
			updatePayload, ok := payload.(*roundevents.ParticipantScoreUpdatedPayload)
			if !ok {
				h.Logger.ErrorContext(ctx, "Invalid payload type for HandleParticipantScoreUpdated",
					attr.Any("payload", payload), // Log the received payload
				)
				return nil, fmt.Errorf("invalid payload type for HandleParticipantScoreUpdated")
			}

			// Use ChannelID from config if payload ChannelID is empty (as a fallback)
			// Or rely solely on payload if backend guarantees population
			channelID := updatePayload.ChannelID
			if channelID == "" && h.Config != nil && h.Config.GetEventChannelID() != "" {
				channelID = h.Config.GetEventChannelID()
			}

			h.Logger.InfoContext(ctx, "Received ParticipantScoreUpdated event", // Updated log message
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", updatePayload.RoundID),
				attr.String("participant_id", string(updatePayload.Participant)),
				attr.Int("score", int(updatePayload.Score)),
				attr.String("event_message_id", updatePayload.EventMessageID),
				// Removed log for participant_count_in_payload as it's no longer expected
			)

			// --- Update the Discord Scorecard Embed using scoreround.UpdateScoreEmbed ---

			// Get the ScoreRoundManager
			scoreRoundManager := h.RoundDiscord.GetScoreRoundManager() // Adjust method call as needed

			// Call UpdateScoreEmbed with the specific user's updated score
			updateResult, err := scoreRoundManager.UpdateScoreEmbed(
				ctx,
				channelID,                    // Pass channel ID
				updatePayload.EventMessageID, // Pass message ID of the scorecard
				updatePayload.Participant,    // Pass the UserID of the updated participant
				&updatePayload.Score,         // Pass a pointer to the updated score
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to call UpdateScoreEmbed", // Updated log message
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
					attr.RoundID("round_id", updatePayload.RoundID),
					attr.String("message_id", updatePayload.EventMessageID),
					attr.String("channel_id", channelID),
					attr.String("user_id", string(updatePayload.Participant)),
				)
				// Decide how to handle this error - maybe publish a Discord update error event
				return nil, fmt.Errorf("failed to call UpdateScoreEmbed: %w", err)
			}
			if updateResult.Error != nil {
				h.Logger.ErrorContext(ctx, "UpdateScoreEmbed returned error in result", // Updated log message
					attr.CorrelationIDFromMsg(msg),
					attr.Error(updateResult.Error),
					attr.RoundID("round_id", updatePayload.RoundID),
					attr.String("message_id", updatePayload.EventMessageID),
					attr.String("channel_id", channelID),
					attr.String("user_id", string(updatePayload.Participant)),
				)
				// Decide how to handle this error - maybe publish a Discord update error event
				return nil, fmt.Errorf("scorecard update failed: %w", updateResult.Error)
			}

			h.Logger.InfoContext(ctx, "Successfully triggered scorecard embed update via UpdateScoreEmbed", // Updated log message
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", updatePayload.RoundID),
				attr.String("message_id", updatePayload.EventMessageID),
				attr.String("channel_id", channelID),
			)

			// --- Handle Score Update Confirmation (Optional) ---
			// This logic would go here, likely using a method on ScoreRoundManager
			// that sends a message to the user (e.g., ephemeral followup or DM).
			// The ParticipantScoreUpdatedPayload might still need the original InteractionID
			// if you're sending ephemeral followups.

			// You can uncomment and adapt your previous confirmation logic here if needed
			// scorePtr := &updatePayload.Score // Get pointer to the score
			// confirmResult, confirmErr := scoreRoundManager.SendScoreUpdateConfirmation(...)
			// ... logging and error handling ...

			// Trace event (optional)
			// tracePayload := map[string]interface{}{...}
			// traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			// return []*message.Message{traceMsg}, nil // Return the trace message

			return nil, nil // Return no messages if the handler's job is just to update Discord
		},
	)(msg) // Execute the wrapped handler
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
