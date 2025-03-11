package discordhandlers

import (
	"context"
	"fmt"
	"log/slog"

	discordevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleSendDM handles the SendUserDM event.
func (h *DiscordHandlers) HandleSendDM(msg *message.Message) ([]*message.Message, error) {
	msg.Metadata.Set("handler_name", "HandleSendDM")
	var payload discordevents.SendDMPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	slog.Info("Sending DM", attr.UserID(payload.UserID), attr.CorrelationIDFromMsg(msg))
	sentMsg, err := h.Discord.SendDM(context.Background(), payload.UserID, payload.Message)
	if err != nil {
		slog.Error("Failed to send DM", attr.Error(err), attr.UserID(payload.UserID), attr.CorrelationIDFromMsg(msg))
		failureResult := h.interactionResponded(msg, payload.UserID, discordevents.StatusFail)
		failureMsg, err := h.Helper.CreateResultMessage(msg, failureResult, discordevents.DMError)
		if err != nil {
			return nil, fmt.Errorf("failed to create failure message: %w", err)
		}
		return []*message.Message{failureMsg}, nil
	}
	// Use attr.DiscordMessageID to log the Discord message ID.
	slog.Info("DM sent successfully", attr.UserID(payload.UserID), attr.DiscordMessageID(sentMsg.ID), attr.DiscordChannelID(sentMsg.ChannelID), attr.CorrelationIDFromMsg(msg))
	successResult := h.interactionResponded(msg, payload.UserID, discordevents.StatusSuccess)
	successMsg, err := h.Helper.CreateResultMessage(msg, successResult, discordevents.DMSent)
	if err != nil {
		return nil, fmt.Errorf("failed to create success message: %w", err)
	}
	return []*message.Message{successMsg}, nil
}
