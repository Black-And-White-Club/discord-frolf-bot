package userhandlers

import (
	"fmt"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoleUpdateCommand handles the /rolerequest command.
func (h *userHandlers) HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateCommand")
	var payload discorduserevents.RoleUpdateCommandPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Validate interaction metadata
	interactionID := msg.Metadata.Get("interaction_id")
	interactionToken := msg.Metadata.Get("interaction_token")
	guildID := msg.Metadata.Get("guild_id")
	if interactionID == "" || interactionToken == "" || guildID == "" {
		err := fmt.Errorf("interaction metadata missing")
		h.Logger.Error(ctx, "Interaction metadata missing", attr.CorrelationIDFromMsg(msg))
		return nil, err
	}

	// Respond to the role request with an ephemeral message
	if err := h.UserDiscord.GetRoleManager().RespondToRoleRequest(ctx, interactionID, interactionToken, payload.TargetUserID); err != nil {
		h.Logger.Error(ctx, "Failed to respond to role request", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to respond to role request: %w", err)
	}

	return nil, nil
}

// HandleRoleUpdateButtonPress handles button interactions.
func (h *userHandlers) HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateButtonPress")
	var payload discorduserevents.RoleUpdateButtonPressPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Extract the selected role from the button's custom ID
	roleStr := strings.TrimPrefix(payload.InteractionCustomID, "role_button_")
	selectedRole := usertypes.UserRoleEnum(roleStr)

	// Acknowledge the button press with an ephemeral message
	err := h.UserDiscord.GetRoleManager().RespondToRoleButtonPress(ctx, payload.InteractionID, payload.InteractionToken, payload.RequesterID, string(selectedRole), payload.TargetUserID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to acknowledge interaction", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to acknowledge interaction: %w", err)
	}

	// Create and publish a backend event for role update
	backendPayload := userevents.UserRoleUpdateRequestPayload{
		RequesterID: usertypes.DiscordID(payload.RequesterID),
		DiscordID:   usertypes.DiscordID(payload.TargetUserID),
		Role:        selectedRole,
	}
	backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserRoleUpdateRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create result message", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to create result message: %w", err)
	}

	// Set metadata for the backend event
	backendEvent.Metadata.Set("interaction_token", payload.InteractionToken)
	backendEvent.Metadata.Set("guild_id", payload.GuildID)

	return []*message.Message{backendEvent}, nil
}

// HandleRoleUpdateResult processes the backend's response.
func (h *userHandlers) HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
	topic := msg.Metadata.Get("topic")
	h.Logger.Info(ctx, "Received role update result", attr.Topic(topic), attr.CorrelationIDFromMsg(msg))

	var payload userevents.UserRoleUpdateResultPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	discordID := string(payload.DiscordID)
	discordRoleID, ok := h.Config.Discord.RoleMappings[string(payload.Role)]
	if !ok {
		err := fmt.Errorf("no Discord role mapping found for application role: %s", payload.Role)
		h.Logger.Error(ctx, "Role mapping error", attr.Error(err))

		err = h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), fmt.Sprintf("Failed to update role: %s", err))
		if err != nil {
			h.Logger.Error(ctx, "Failed to send ephemeral message", attr.Error(err))
			return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
		}
		return nil, nil
	}

	if !payload.Success {
		err := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), fmt.Sprintf("Failed to update role: %s", payload.Error))
		if err != nil {
			h.Logger.Error(ctx, "Failed to send ephemeral message", attr.Error(err))
			return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
		}
		return nil, nil
	}

	guildID := msg.Metadata.Get("guild_id")
	if guildID == "" {
		err := fmt.Errorf("guild ID missing from message metadata")
		h.Logger.Error(ctx, "Guild ID missing", attr.Error(err))

		err = h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), fmt.Sprintf("Failed to update role: %s", err))
		if err != nil {
			h.Logger.Error(ctx, "Failed to send ephemeral message", attr.Error(err))
			return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
		}
		return nil, nil
	}

	// Add the role to the user in Discord
	err := h.UserDiscord.GetRoleManager().AddRoleToUser(ctx, guildID, discordID, discordRoleID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))

		err = h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), fmt.Sprintf("Role updated in application, but failed to sync with Discord: %s", err))
		if err != nil {
			h.Logger.Error(ctx, "Failed to send ephemeral message", attr.Error(err))
			return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
		}
		return nil, nil
	}

	// Send ephemeral message for successful update
	err = h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), "Role update completed")
	if err != nil {
		h.Logger.Error(ctx, "Failed to send ephemeral message", attr.Error(err))
		return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
	}

	return nil, nil
}

// HandleAddRole handles the AddRole event (currently from the signup flow)
func (h *userHandlers) HandleAddRole(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	var payload discorduserevents.AddRolePayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.Error(err))
		return nil, nil
	}

	// Attempt to add the role
	err := h.UserDiscord.GetRoleManager().AddRoleToUser(ctx, h.Config.Discord.GuildID, string(payload.DiscordID), payload.RoleID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))
		// Create a failure message
		failurePayload := discorduserevents.RoleAdditionFailedPayload{
			DiscordID: payload.DiscordID,
			Reason:    err.Error(),
		}
		failureMsg, createErr := h.Helper.CreateResultMessage(msg, failurePayload, discorduserevents.SignupRoleAdditionFailed)
		if createErr != nil {
			h.Logger.Error(ctx, "Failed to create failure message", attr.Error(createErr))
			return nil, fmt.Errorf("failed to create failure message: %w", createErr)
		}
		return []*message.Message{failureMsg}, nil
	}

	// If successful, create a success message
	successPayload := discorduserevents.RoleAddedPayload{
		DiscordID: payload.DiscordID,
	}
	successMsg, createErr := h.Helper.CreateResultMessage(msg, successPayload, discorduserevents.SignupRoleAdded)
	if createErr != nil {
		h.Logger.Error(ctx, "Failed to create success message", attr.Error(createErr))
		return nil, fmt.Errorf("failed to create success message: %w", createErr)
	}

	return []*message.Message{successMsg}, nil
}
