package userhandlers

import (
	"fmt"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserSignupRequest handles the UserSignupRequest event from the Discord bot.
func (h *userHandlers) HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserSignupRequest")
	var payload userevents.UserSignupRequestPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	backendPayload := userevents.UserSignupRequestPayload{
		DiscordID: payload.DiscordID,
		TagNumber: payload.TagNumber,
	}
	backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserSignupRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create backend event", attr.Error(err))
		return nil, fmt.Errorf("failed to create backend event: %w", err)
	}
	return []*message.Message{backendEvent}, nil
}

// HandleUser Created handles the UserCreated event from the backend.
func (h *userHandlers) HandleUserCreated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	var payload userevents.UserCreatedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	rolePayload := discorduserevents.AddRolePayload{
		DiscordID: string(payload.DiscordID),
		RoleID:    h.Config.Discord.RegisteredRoleID,
	}
	roleMsg, err := h.Helper.CreateResultMessage(msg, rolePayload, discorduserevents.SignupAddRole)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create add role event", attr.Error(err))
		return nil, fmt.Errorf("failed to create add role event: %w", err)
	}
	return []*message.Message{roleMsg}, nil
}

// HandleUser CreationFailed handles the UserCreationFailed event from the backend.
func (h *userHandlers) HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUser CreationFailed")
	var payload userevents.UserCreationFailedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Use the signup manager to send the failure result
	correlationID := msg.Metadata.Get("correlation_id")
	h.UserDiscord.GetSignupManager().SendSignupResult(correlationID, false) // Indicate failure
	h.Logger.Error(ctx, "User creation failed", attr.String("reason", payload.Reason))

	return nil, nil
}

// HandleRoleAdded handles the RoleAdded event.
func (h *userHandlers) HandleRoleAdded(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	var payload discorduserevents.RoleAddedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, nil
	}

	// Update the interaction response to indicate success
	correlationID := msg.Metadata.Get("correlation_id")
	h.UserDiscord.GetSignupManager().SendSignupResult(correlationID, true)

	return nil, nil
}

// HandleRoleAdditionFailed handles the RoleAdditionFailed event.
func (h *userHandlers) HandleRoleAdditionFailed(msg *message.Message) ([]*message.Message, error) {
	var payload discorduserevents.RoleAdditionFailedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Update the interaction response to indicate failure
	correlationID := msg.Metadata.Get("correlation_id")
	h.UserDiscord.GetSignupManager().SendSignupResult(correlationID, false)

	return nil, nil
}
