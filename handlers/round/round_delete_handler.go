package roundhandlers

import (
	"encoding/json"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundDeleted handles the round.deleted event.
func (h *RoundHandlers) HandleRoundDeleted(msg *message.Message) error {
	var payload roundevents.RoundDeletedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.logger.Error("Failed to unmarshal RoundDeleted payload", "error", err)
		return err
	}

	userID, err := extractUserID(msg.Metadata.Get("correlation_id"))
	if err != nil {
		h.logger.Error("Failed to extract user_id", "error", err)
		return err
	}

	// Notify users about the round deletion
	h.sendUserMessage(userID, "A round has been deleted.", msg.Metadata.Get("correlation_id"))

	h.logger.Info("Handled round.deleted event", "round_id", payload.RoundID)

	return nil
}
