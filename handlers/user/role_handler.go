package userhandlers

import (
	"fmt"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoleUpdateCommand handles the /rolerequest command.
func (h *UserHandlers) HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateCommand")

	var payload discorduserevents.RoleUpdateCommandPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interactionID := msg.Metadata.Get("interaction_id")
	interactionToken := msg.Metadata.Get("interaction_token")
	guildID := msg.Metadata.Get("guild_id")
	if interactionID == "" || interactionToken == "" || guildID == "" {
		err := fmt.Errorf("interaction metadata missing")
		h.Logger.Error(ctx, "Interaction metadata missing", attr.CorrelationIDFromMsg(msg))
		return nil, err
	}

	if err := h.Discord.RespondToRoleRequest(ctx, interactionID, interactionToken, payload.TargetUserID); err != nil {
		h.Logger.Error(ctx, "Failed to respond to role request", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to respond to role request: %w", err)
	}

	return nil, nil
}

// HandleRoleUpdateButtonPress handles button interactions.
func (h *UserHandlers) HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateButtonPress")
	var payload discorduserevents.RoleUpdateButtonPressPayload

	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	roleStr := strings.TrimPrefix(payload.InteractionCustomID, "role_button_")
	selectedRole := usertypes.UserRoleEnum(roleStr)

	err := h.Discord.RespondToRoleButtonPress(ctx, payload.InteractionID, payload.InteractionToken, payload.RequesterID, string(selectedRole), payload.TargetUserID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to acknowledge interaction", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to acknowledge interaction: %w", err)
	}

	backendPayload := userevents.UserRoleUpdateRequestPayload{
		RequesterID: payload.RequesterID,
		DiscordID:   usertypes.DiscordID(payload.TargetUserID),
		Role:        selectedRole,
	}
	backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserRoleUpdateRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create result message", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to create result message: %w", err)
	}
	backendEvent.Metadata.Set("interaction_token", payload.InteractionToken)
	backendEvent.Metadata.Set("guild_id", payload.GuildID) // Pass GuildID

	return []*message.Message{backendEvent}, nil
}

// HandleRoleUpdateResult processes the backend's response.
func (h *UserHandlers) HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
	topic := msg.Metadata.Get("topic")
	h.Logger.Info(ctx, "Received role update result", attr.Topic(topic), attr.CorrelationIDFromMsg(msg))

	var payload userevents.UserRoleUpdateResultPayload
	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interactionToken := msg.Metadata.Get("interaction_token")
	guildID := msg.Metadata.Get("guild_id") // Get Guild ID
	if interactionToken == "" || guildID == "" {
		err := fmt.Errorf("interaction_token or guild_id missing from metadata")
		h.Logger.Error(ctx, "interaction_token or guild_id missing from metadata", attr.Error(err))
		return nil, err
	}

	content := "Role update completed"
	// Get Discord Role ID from config (or database)
	discordRoleID, ok := h.Config.Discord.RoleMappings[string(payload.Role)]
	if !ok {
		err := fmt.Errorf("no Discord role mapping found for application role: %s", payload.Role)
		h.Logger.Error(ctx, "Role mapping error", attr.Error(err))
		content = fmt.Sprintf("Failed to update role: %s", err)
		if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, content); err != nil {
			h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
			return nil, fmt.Errorf("failed to edit interaction response: %w", err)
		}
		return nil, err
	}
	if !payload.Success {
		content = fmt.Sprintf("Failed to update role: %s", payload.Error)
		if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, content); err != nil {
			h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
			return nil, fmt.Errorf("failed to edit interaction response: %w", err)
		}
		return nil, nil
	}

	// Add the Discord role.
	err := h.Discord.AddRoleToUser(ctx, guildID, string(payload.DiscordID), discordRoleID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))
		// Send a follow-up message indicating the Discord role sync failed.
		content = fmt.Sprintf("Role updated in application, but failed to sync with Discord: %s", err)
	}

	if err := h.Discord.EditRoleUpdateResponse(ctx, interactionToken, content); err != nil {
		h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
		return nil, fmt.Errorf("failed to edit interaction response: %w", err)
	}

	return nil, nil
}
