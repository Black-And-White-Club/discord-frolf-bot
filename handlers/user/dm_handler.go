package userhandlers

import (
	"fmt"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/events" // Import events
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleSendUserDM handles the SendUserDM event.
func (h *UserHandlers) HandleSendUserDM(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleSendUserDM")

	var payload discorduserevents.SendUserDMPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil { // Use the helper
		return nil, err
	}

	h.Logger.Info(msg.Context(), "Sending DM", attr.UserID(payload.UserID), attr.CorrelationIDFromMsg(msg))

	channel, err := h.Session.UserChannelCreate(payload.UserID)
	if err != nil {
		return h.handleDMError(msg.Context(), msg, payload.UserID, discorduserevents.DMCreateError, fmt.Errorf("failed to create DM channel: %w", err))
	}

	if _, err = h.Session.ChannelMessageSend(channel.ID, payload.Message); err != nil {
		return h.handleDMError(msg.Context(), msg, payload.UserID, discorduserevents.DMSendError, fmt.Errorf("failed to send DM: %w", err))
	}

	h.Logger.Info(msg.Context(), "DM sent successfully", attr.UserID(payload.UserID), attr.String("channel_id", channel.ID), attr.CorrelationIDFromMsg(msg))

	successPayload := discorduserevents.DMSentPayload{
		UserID:    payload.UserID,
		ChannelID: channel.ID,
		CommonMetadata: events.CommonMetadata{
			EventName: discorduserevents.DMSent,
			Domain:    userDomain,
		},
	}

	successMsg, err := h.createResultMessage(msg, successPayload)
	if err != nil {
		return nil, err
	}
	return []*message.Message{successMsg}, nil
}

// HandleDMSent handles the DMSent event.
func (h *UserHandlers) HandleDMSent(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleDMSent")
	var payload discorduserevents.DMSentPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	h.Logger.Info(msg.Context(), "DM sent successfully (confirmation received)",
		attr.UserID(payload.UserID),
		attr.String("channel_id", payload.ChannelID),
		attr.CorrelationIDFromMsg(msg),
	)

	successResult := h.interactionResponded(msg, payload.UserID, interactionSuccess, "", discorduserevents.DMSent)
	successMsg, err := h.createResultMessage(msg, successResult)
	if err != nil {
		return nil, err
	}
	return []*message.Message{successMsg}, nil
}

// HandleDMCreateError handles errors creating the DM channel.
func (h *UserHandlers) HandleDMCreateError(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleDMCreateError")
	var payload discorduserevents.DMErrorPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}
	return h.handleDMFailure(msg, discorduserevents.DMCreateError, payload)
}

// HandleDMSendError handles errors sending the DM.
func (h *UserHandlers) HandleDMSendError(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleDMSendError")
	var payload discorduserevents.DMErrorPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}
	// Now uses the helper function.  Much cleaner!
	return h.handleDMFailure(msg, discorduserevents.DMSendError, payload)
}

// handleDMFailure handles both DM creation and sending failures.
func (h *UserHandlers) handleDMFailure(msg *message.Message, eventName string, payload discorduserevents.DMErrorPayload) ([]*message.Message, error) {
	h.Logger.Error(msg.Context(), "DM operation failed",
		attr.Error(fmt.Errorf("%s", payload.ErrorDetail)),
		attr.UserID(payload.UserID),
		attr.CorrelationIDFromMsg(msg),
	)
	failureResult := h.interactionResponded(msg, payload.UserID, interactionFailure, payload.ErrorDetail, eventName)
	failureMsg, err := h.createResultMessage(msg, failureResult)
	if err != nil {
		return nil, err
	}
	return []*message.Message{failureMsg}, nil
}
