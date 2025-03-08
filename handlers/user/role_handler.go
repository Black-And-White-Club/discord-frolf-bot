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
		RequesterID: usertypes.DiscordID(payload.RequesterID),
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
	// Extract the Discord ID from the payload
	discordID := string(payload.DiscordID)
	// Get Discord Role ID from config (or database)
	discordRoleID, ok := h.Config.Discord.RoleMappings[string(payload.Role)]
	if !ok {
		err := fmt.Errorf("no Discord role mapping found for application role: %s", payload.Role)
		h.Logger.Error(ctx, "Role mapping error", attr.Error(err))
		// Send DM to user about the failure
		dmMsg, dmErr := h.createDMMessage(ctx, discordID, fmt.Sprintf("Failed to update role: %s", err))
		if dmErr != nil {
			h.Logger.Error(ctx, "Failed to create DM message", attr.Error(dmErr))
			return nil, fmt.Errorf("failed to create DM message: %w", dmErr)
		}
		return []*message.Message{dmMsg}, nil // Return the DM message
	}
	if !payload.Success {
		// Send DM to user about the failure
		dmMsg, dmErr := h.createDMMessage(ctx, discordID, fmt.Sprintf("Failed to update role: %s", payload.Error))
		if dmErr != nil {
			h.Logger.Error(ctx, "Failed to create DM message", attr.Error(dmErr))
			return nil, fmt.Errorf("failed to create DM message: %w", dmErr)
		}
		return []*message.Message{dmMsg}, nil // Return the DM message
	}
	// Add the Discord role (if the update was successful)
	err := h.Discord.AddRoleToUser(ctx, msg.Metadata.Get("guild_id"), discordID, discordRoleID)
	if err != nil {
		h.Logger.Error(ctx, "Failed to add Discord role", attr.Error(err))
		// Send a DM indicating the Discord role sync failed.
		dmMsg, dmErr := h.createDMMessage(ctx, discordID, fmt.Sprintf("Role updated in application, but failed to sync with Discord: %s", err))
		if dmErr != nil {
			h.Logger.Error(ctx, "Failed to create DM message", attr.Error(dmErr))
			return nil, fmt.Errorf("failed to create DM message: %w", dmErr)
		}
		return []*message.Message{dmMsg}, nil // Return the DM message
	}
	// Send DM to user about the success
	dmMsg, dmErr := h.createDMMessage(ctx, discordID, "Role update completed")
	if dmErr != nil {
		h.Logger.Error(ctx, "Failed to create DM message", attr.Error(dmErr))
		return nil, fmt.Errorf("failed to create DM message: %w", dmErr)
	}
	return []*message.Message{dmMsg}, nil
}
