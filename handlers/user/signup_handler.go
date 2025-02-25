package userhandlers

import (
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserSignupRequest handles the UserSignupRequest event from the Discord bot.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserSignupRequest")

	var payload userevents.UserSignupRequestPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Create a new event for the backend.  We're *transforming* the event.
	backendPayload := userevents.UserSignupRequestPayload{
		DiscordID: payload.DiscordID,
		TagNumber: payload.TagNumber,
	}
	backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserSignupRequest) //Forward the event.
	if err != nil {
		h.Logger.Error(ctx, "Failed to create backend event", attr.Error(err))
		return nil, fmt.Errorf("failed to create backend event: %w", err)
	}
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
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interactionToken := msg.Metadata.Get("interaction_token")
	guildID := msg.Metadata.Get("guild_id")

	if interactionToken == "" || guildID == "" {
		err := fmt.Errorf("interaction_token or guild_id missing from metadata")
		h.Logger.Error(ctx, "interaction_token or guild_id missing from metadata", attr.Error(err))
		return nil, err
	}

	// 1. Add the "Rattler" role.
	discordRoleID, ok := h.Config.Discord.RoleMappings["rattler"]
	if !ok {
		err := fmt.Errorf("no Discord role mapping found for application role: rattler")
		h.Logger.Error(ctx, "Role mapping error", attr.Error(err))
		content := fmt.Sprintf("Failed to update role: %s", err)
		//Need to edit interaction
		if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, content); err != nil {
			h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
			return nil, fmt.Errorf("failed to edit interaction response: %w", err)
		}
		return nil, err
	}

	if err := h.Discord.AddRoleToUser(ctx, guildID, string(payload.DiscordID), discordRoleID); err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))
		content := fmt.Sprintf("Signup succeeded, but failed to sync Discord role: %s. Contact an admin", err)
		h.Discord.EditRoleUpdateResponse(ctx, interactionToken, content)
		return nil, fmt.Errorf("failed to add role: %w", err)
	}
	// 2. Update the original interaction response.
	successMsg := "Signup complete! You now have access to the members-only channels."
	if payload.TagNumber != nil {
		successMsg = fmt.Sprintf("Signup complete! Your tag number is %d. You now have access to the members-only channels.", *payload.TagNumber)
	}

	if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, successMsg); err != nil {
		h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
		return nil, fmt.Errorf("failed to edit interaction response: %w", err)
	}
	return nil, nil
}

// HandleUserCreationFailed handles the UserCreationFailed event from the backend.
func (h *UserHandlers) HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleUserCreationFailed")

	var payload userevents.UserCreationFailedPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interactionToken := msg.Metadata.Get("interaction_token")

	if interactionToken == "" {
		err := fmt.Errorf("interaction_token missing from metadata")
		h.Logger.Error(ctx, "interaction_token missing from metadata", attr.Error(err))
		return nil, err
	}

	// Update the original interaction response.
	failMsg := fmt.Sprintf("Signup failed: %s. Please try again by reacting to the message in the signup channel, or contact an administrator.", payload.Reason)
	if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, failMsg); err != nil {
		h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
		return nil, fmt.Errorf("failed to edit interaction response: %w", err)
	}

	return nil, nil
}
