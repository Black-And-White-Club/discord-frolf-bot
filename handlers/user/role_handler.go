package userhandlers

import (
	"fmt"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// allowedRoles defines the set of valid roles.  Consider moving this to a config file or constants.
var allowedRoles = map[usertypes.UserRoleEnum]bool{
	usertypes.UserRoleRattler: true,
	usertypes.UserRoleEditor:  true,
	usertypes.UserRoleAdmin:   true,
}

// HandleRoleUpdateCommand handles the /updaterole command.
func (h *UserHandlers) HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateCommand")

	var payload discorduserevents.RoleUpdateCommandPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Get interaction metadata from the *original* message (important for correlation).
	interactionID := msg.Metadata.Get("interaction_id")
	interactionToken := msg.Metadata.Get("interaction_token")
	if interactionID == "" || interactionToken == "" {
		err := fmt.Errorf("interaction metadata missing")
		h.Logger.Error(ctx, "Interaction metadata missing", attr.CorrelationIDFromMsg(msg))
		return nil, err
	}

	// Create buttons for roles.
	var buttons []discordgo.MessageComponent
	for role := range allowedRoles {
		buttons = append(buttons, discordgo.Button{
			Label:    string(role),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("role_button_%s", role),
		})
	}
	buttons = append(buttons, discordgo.Button{
		Label:    "Cancel",
		Style:    discordgo.DangerButton,
		CustomID: "role_button_cancel",
	})

	h.Logger.Info(ctx, "Responding to interaction", attr.CorrelationIDFromMsg(msg))
	// Respond to the interaction (show the buttons).
	err := h.interactionRespond(&discordgo.Interaction{ID: interactionID, Token: interactionToken}, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Please choose a role for <@%s>:", payload.TargetUserID),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: buttons},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		h.Logger.Error(ctx, "Failed to send interaction response", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to send interaction response: %w", err) // Return error, Watermill will Nack
	}

	return nil, nil
}

// HandleRoleUpdateButtonPress handles button interactions.
func (h *UserHandlers) HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateButtonPress")
	var payload discorduserevents.RoleUpdateButtonPressPayload

	// Unmarshal the payload *before* using its values.
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Parse custom ID (still needed, but simplified).
	roleStr := strings.TrimPrefix(payload.InteractionID, "role_button_")
	selectedRole := usertypes.UserRoleEnum(roleStr)

	// Acknowledge the interaction with a message update.
	updateMsg := fmt.Sprintf("<@%s> has requested role '%s' for <@%s>. Request is being processed.",
		payload.RequesterID, selectedRole, payload.TargetUserID)

	//Use our interactionRespond helper
	err := h.interactionRespond(&discordgo.Interaction{ID: payload.InteractionID, Token: payload.InteractionToken}, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    updateMsg,
			Components: []discordgo.MessageComponent{},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		h.Logger.Error(ctx, "Failed to acknowledge interaction", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to acknowledge interaction: %w", err)
	}

	// Publish the event to the backend for processing.
	backendPayload := userevents.UserRoleUpdateRequestPayload{
		RequesterID: payload.RequesterID,
		DiscordID:   usertypes.DiscordID(payload.TargetUserID),
		Role:        selectedRole,
	}
	//Create event to publish and return
	backendEvent, err := h.createResultMessage(msg, backendPayload)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create result message", attr.Error(err), attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("failed to create result message: %w", err)
	}
	backendEvent.Metadata.Set("interaction_token", payload.InteractionToken) // Pass token for later update
	return []*message.Message{backendEvent}, nil
}

// HandleRoleUpdateResult processes the backend's response.
func (h *UserHandlers) HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
	topic := msg.Metadata.Get("topic")
	h.Logger.Info(ctx, "Received role update result", attr.Topic(topic), attr.CorrelationIDFromMsg(msg))

	var payload userevents.UserRoleUpdateResultPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interactionToken := msg.Metadata.Get("interaction_token")
	if interactionToken == "" {
		err := fmt.Errorf("interaction_token missing from metadata")
		h.Logger.Error(ctx, "interaction_token missing from metadata", attr.Error(err))
		return nil, err
	}

	content := "Role update completed"
	if topic == userevents.UserRoleUpdateFailed {
		content = fmt.Sprintf("Failed to update role: %s", payload.Error)
	}

	// Edit the original interaction response - pass token directly
	_, err := h.Session.InteractionResponseEdit(
		&discordgo.Interaction{Token: interactionToken}, // Wrap token in Interaction struct
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		h.Logger.Error(ctx, "Failed to edit interaction response", attr.Error(err))
		return nil, fmt.Errorf("failed to edit interaction response: %w", err)
	}

	return nil, nil
}
