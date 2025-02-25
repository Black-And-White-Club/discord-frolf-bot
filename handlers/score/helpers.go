package scorehandlers

import (
	"encoding/json"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// createResultMessage creates a new Watermill message and sets metadata.
func (h *ScoreHandlers) createResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	// Copy metadata from the original message.  Watermill *will* propagate the correlation ID.
	if originalMsg != nil {
		for key, value := range originalMsg.Metadata {
			newEvent.Metadata.Set(key, value)
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.Logger.Error(originalMsg.Context(), "Failed to marshal payload in createResultMessage", attr.Error(err), attr.CorrelationIDFromMsg(originalMsg))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	newEvent.Payload = payloadBytes

	newEvent.Metadata.Set("handler_name", "createResultMessage")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "score")

	return newEvent, nil
}

// unmarshalPayload is a generic helper to unmarshal message payloads.
func (h *ScoreHandlers) unmarshalPayload(msg *message.Message, payload interface{}) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		h.Logger.Error(msg.Context(), "Failed to unmarshal payload", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return nil
}
