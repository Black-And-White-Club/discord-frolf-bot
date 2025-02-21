package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

const (
	userDomain                = "discord_user"
	interactionRespondedTopic = "discord.interaction.responded"
	interactionSuccess        = "success"
	interactionFailure        = "failure"
)

// handleDMError is a helper function to handle DM-related errors.
func (h *UserHandlers) handleDMError(ctx context.Context, msg *message.Message, userID string, eventName string, originalErr error) ([]*message.Message, error) {
	h.Logger.Error(ctx, "DM operation failed", attr.Error(originalErr), attr.UserID(userID), attr.CorrelationIDFromMsg(msg))

	errorPayload := discorduserevents.DMErrorPayload{
		UserID:      userID,
		ErrorDetail: originalErr.Error(), // Use err.Error() directly
		CommonMetadata: events.CommonMetadata{
			EventName: eventName,
			Domain:    userDomain,
			Timestamp: time.Now(), // Set the timestamp here
		},
	}

	errMsg, createErr := h.createResultMessage(msg, errorPayload)
	if createErr != nil {
		return nil, createErr // Return createResultMessage error
	}
	return []*message.Message{errMsg}, originalErr // Return original error
}

// interactionResponded creates a payload for Discord interaction responses.
func (h *UserHandlers) interactionResponded(msg *message.Message, userID, status, errorDetail, eventName string) discorduserevents.InteractionRespondedPayload {
	interactionPayload := discorduserevents.InteractionRespondedPayload{
		InteractionID: msg.Metadata.Get("interaction_id"),
		UserID:        userID,
		Status:        status,
		ErrorDetail:   errorDetail,
		CommonMetadata: events.CommonMetadata{ // Use CommonMetadata
			Domain:    userDomain,
			EventName: eventName, // Use the passed eventName
			Timestamp: time.Now(),
		},
	}

	return interactionPayload
}

// newResultPayload creates a ResultPayload (generic).  This might not be strictly needed anymore.
func (h *UserHandlers) newResultPayload(status, errorDetail, eventName string) events.ResultPayload {
	return events.ResultPayload{
		CommonMetadata: events.CommonMetadata{
			Domain:    userDomain,
			EventName: fmt.Sprintf("%s.%s", userDomain, eventName),
			Timestamp: time.Now(),
		},
		Status:      status,
		ErrorDetail: errorDetail,
	}
}

// createResultMessage creates a new Watermill message and sets metadata.
func (h *UserHandlers) createResultMessage(originalMsg *message.Message, payload interface{}) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	// Copy metadata from the original message.
	if originalMsg != nil {
		for key, value := range originalMsg.Metadata {
			newEvent.Metadata.Set(key, value)
		}
	}

	// Use the shared WithMetadata.  Handle errors.
	if err := events.WithMetadata(newEvent, payload, h.Logger); err != nil {
		h.Logger.Error(newEvent.Context(), "Failed to marshal payload in createResultMessage", attr.Error(err))
		return nil, err
	}

	// Set topic *only* for InteractionRespondedPayload.
	if _, ok := payload.(discorduserevents.InteractionRespondedPayload); ok {
		newEvent.Metadata.Set("topic", interactionRespondedTopic)
	}

	return newEvent, nil
}

// interactionRespond is a helper to respond to Discord interactions.
func (h *UserHandlers) interactionRespond(interaction *discordgo.Interaction, response *discordgo.InteractionResponse) error {
	return h.Session.InteractionRespond(interaction, response)
}

// unmarshalPayload is a generic helper function to unmarshal the message payload.
func (h *UserHandlers) unmarshalPayload(msg *message.Message, payload interface{}) error {
	if err := json.Unmarshal(msg.Payload, payload); err != nil {
		h.Logger.Error(msg.Context(), "Failed to unmarshal payload", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return nil
}
