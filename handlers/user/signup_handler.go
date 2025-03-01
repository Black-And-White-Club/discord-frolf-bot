package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserSignupRequest handles the UserSignupRequest event from the Discord bot.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserSignupRequest")

	var payload userevents.UserSignupRequestPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Transform event
	backendPayload := userevents.UserSignupRequestPayload{
		DiscordID: payload.DiscordID,
		TagNumber: payload.TagNumber,
	}
	backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserSignupRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create backend event", attr.Error(err))
		return nil, fmt.Errorf("failed to create backend event: %w", err)
	}

	// Preserve important metadata
	backendEvent.Metadata.Set("interaction_id", msg.Metadata.Get("interaction_id"))
	backendEvent.Metadata.Set("interaction_token", msg.Metadata.Get("interaction_token"))
	backendEvent.Metadata.Set("guild_id", msg.Metadata.Get("guild_id"))

	return []*message.Message{backendEvent}, nil
}

// HandleUserCreated handles the UserCreated event from the backend.
func (h *UserHandlers) HandleUserCreated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserCreated")

	var payload userevents.UserCreatedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, nil
	}

	userID := string(payload.DiscordID)
	h.Logger.Info(ctx, "Adding role to user", attr.String("discord_id", userID))

	err := h.Discord.AddRoleToUser(ctx, h.Config.Discord.GuildID, userID, h.Config.Discord.RegisteredRoleID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))
		failureMsg := fmt.Sprintf("Signup successful, but failed to sync Discord role: %s. Contact an admin.", err)
		dmMsg, err := h.createDMMessage(ctx, userID, failureMsg)
		if err != nil {
			h.Logger.Error(ctx, "Failed to create DM message", attr.Error(err))
			return nil, err
		}
		return []*message.Message{dmMsg}, nil
	}

	successMsg := "Signup complete! You now have access to the members-only channels."
	if payload.TagNumber != nil {
		successMsg = fmt.Sprintf("Signup complete! Your tag number is %d. You now have access to the members-only channels.", *payload.TagNumber)
	}

	dmMsg, err := h.createDMMessage(ctx, userID, successMsg)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Created DM message",
		attr.String("user_id", userID),
		attr.String("message_id", dmMsg.UUID),
	)

	return []*message.Message{dmMsg}, nil
}

// HandleUserCreationFailed handles the UserCreationFailed event from the backend.
func (h *UserHandlers) HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserCreationFailed")

	var payload userevents.UserCreationFailedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, nil
	}

	userID := string(payload.DiscordID)

	failMsg := fmt.Sprintf("Signup failed: %s. Please try again by reacting to the message in the signup channel, or contact an administrator.", payload.Reason)
	dmMsg, err := h.createDMMessage(ctx, userID, failMsg)
	if err != nil {
		return nil, err
	}

	return []*message.Message{dmMsg}, nil
}

// createDMMessage creates a DM message.
func (h *UserHandlers) createDMMessage(ctx context.Context, userID, messageContent string) (*message.Message, error) {
	payload := discorduserevents.SendUserDMPayload{
		UserID:  userID,
		Message: messageContent,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	dmMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	dmMsg.Metadata.Set("topic", discorduserevents.SendUserDM)

	h.Logger.Info(ctx, "CreatedDM message",
		attr.String("user_id", userID),
		attr.String("message_id", dmMsg.UUID),
	)

	return dmMsg, nil
}
