package roundhandlers

import (
	"encoding/json"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundUpdated handles the round.schedule.update event which fires after a round is updated
func (h *RoundHandlers) HandleRoundUpdated(msg *message.Message) error {
	var payload roundevents.RoundScheduleUpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.logger.Error("Failed to unmarshal RoundScheduleUpdate payload", "error", err)
		return err
	}

	userID, err := extractUserID(msg.Metadata.Get("correlation_id"))
	if err != nil {
		h.logger.Error("Failed to extract user_id", "error", err)
		return err
	}

	// Notify users about the schedule update
	h.sendUserMessage(userID, "The schedule for a round has been updated.", msg.Metadata.Get("correlation_id"))

	h.logger.Info("Handled round.schedule.update event", "round_id", payload.RoundID)

	return nil
}
