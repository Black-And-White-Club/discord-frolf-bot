package userhandlers

import (
	"context"
	"fmt"

	discordevents "github.com/Black-And-White-Club/discord-frolf-bot/events/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleSendUserDM handles the SendUserDM event.
func (h *UserHandlers) HandleSendUserDM(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleSendUserDM")

	var payload discorduserevents.SendUserDMPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	h.Logger.Info(msg.Context(), "Sending DM", attr.UserID(payload.UserID), attr.CorrelationIDFromMsg(msg))

	sentMsg, err := h.Discord.SendDM(context.Background(), payload.UserID, payload.Message)
	if err != nil {
		h.Logger.Error(msg.Context(), "Failed to send DM", attr.Error(err), attr.UserID(payload.UserID), attr.CorrelationIDFromMsg(msg))

		failureResult := h.interactionResponded(msg, payload.UserID, discordevents.StatusFail)
		failureMsg, err := h.Helper.CreateResultMessage(msg, failureResult, discorduserevents.DMError)
		if err != nil {
			return nil, fmt.Errorf("failed to create failure message: %w", err)
		}
		return []*message.Message{failureMsg}, nil
	}

	// Use attr.DiscordMessageID to log the Discord message ID.
	h.Logger.Info(msg.Context(), "DM sent successfully", attr.UserID(payload.UserID), attr.DiscordMessageID(sentMsg.ID), attr.DiscordChannelID(sentMsg.ChannelID), attr.CorrelationIDFromMsg(msg))

	successResult := h.interactionResponded(msg, payload.UserID, discordevents.StatusSuccess)
	successMsg, err := h.Helper.CreateResultMessage(msg, successResult, discorduserevents.DMSent)
	if err != nil {
		return nil, fmt.Errorf("failed to create success message: %w", err)
	}
	return []*message.Message{successMsg}, nil
}
