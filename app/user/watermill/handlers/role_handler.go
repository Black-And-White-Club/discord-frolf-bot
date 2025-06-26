package userhandlers

import (
	"context"
	"fmt"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoleUpdateCommand handles the /rolerequest command.
func (h *UserHandlers) HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error) {
	interactionID := msg.Metadata.Get("interaction_id")
	interactionToken := msg.Metadata.Get("interaction_token")
	guildID := msg.Metadata.Get("guild_id")
	if interactionID == "" || interactionToken == "" || guildID == "" {
		err := fmt.Errorf("interaction metadata missing")
		h.Logger.ErrorContext(context.Background(), "Interaction metadata missing", attr.CorrelationIDFromMsg(msg)) // Use a background context here as the handler's context isn't yet available
		return nil, err
	}

	return h.handlerWrapper(
		"HandleRoleUpdateCommand",
		&discorduserevents.RoleUpdateCommandPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			cmdPayload := payload.(*discorduserevents.RoleUpdateCommandPayload)

			// Respond to the role request with an ephemeral message
			result, err := h.UserDiscord.GetRoleManager().RespondToRoleRequest(ctx, interactionID, interactionToken, cmdPayload.TargetUserID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to respond to role request", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to respond to role request: %w", err)
			}

			// Log operation result if needed
			h.Logger.InfoContext(ctx, "Role request response sent",
				attr.Any("result", result),
				attr.CorrelationIDFromMsg(msg))

			return nil, nil
		},
	)(msg)
}

// HandleRoleUpdateButtonPress handles button interactions.
func (h *UserHandlers) HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleUpdateButtonPress",
		&discorduserevents.RoleUpdateButtonPressPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			buttonPayload := payload.(*discorduserevents.RoleUpdateButtonPressPayload)

			// Extract the selected role from the button's custom ID
			roleStr := strings.TrimPrefix(buttonPayload.InteractionCustomID, "role_button_")
			selectedRole := sharedtypes.UserRoleEnum(roleStr)

			// Acknowledge the button press with an ephemeral message
			result, err := h.UserDiscord.GetRoleManager().RespondToRoleButtonPress(ctx, buttonPayload.InteractionID,
				buttonPayload.InteractionToken, sharedtypes.DiscordID(buttonPayload.RequesterID), string(selectedRole), sharedtypes.DiscordID(buttonPayload.TargetUserID))
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to acknowledge interaction", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to acknowledge interaction: %w", err)
			}

			// Log result status
			h.Logger.InfoContext(ctx, "Role button press response sent",
				attr.Any("result", result),
				attr.CorrelationIDFromMsg(msg))

			// Create and publish a backend event for role update
			backendPayload := userevents.UserRoleUpdateRequestPayload{
				RequesterID: sharedtypes.DiscordID(buttonPayload.RequesterID),
				UserID:      sharedtypes.DiscordID(buttonPayload.TargetUserID),
				Role:        selectedRole,
			}
			backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserRoleUpdateRequest)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err), attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to create result message: %w", err)
			}

			// Set metadata for the backend event
			backendEvent.Metadata.Set("interaction_token", buttonPayload.InteractionToken)
			backendEvent.Metadata.Set("guild_id", buttonPayload.GuildID)

			return []*message.Message{backendEvent}, nil
		},
	)(msg)
}

// HandleRoleUpdateResult processes the backend's response.
func (h *UserHandlers) HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleUpdateResult",
		&userevents.UserRoleUpdateResultPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			resultPayload := payload.(*userevents.UserRoleUpdateResultPayload)
			topic := msg.Metadata.Get("topic")
			h.Logger.InfoContext(ctx, "Received role update result", attr.Topic(topic), attr.CorrelationIDFromMsg(msg))

			discordID := string(resultPayload.UserID)
			discordRoleID, ok := h.Config.GetRoleMappings()[string(resultPayload.Role)]
			if !ok {
				err := fmt.Errorf("no Discord role mapping found for application role: %s", resultPayload.Role)
				h.Logger.ErrorContext(ctx, "Role mapping error", attr.Error(err))

				result, editErr := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"),
					fmt.Sprintf("Failed to update role: %s", err))
				if editErr != nil {
					h.Logger.ErrorContext(ctx, "Failed to send ephemeral message", attr.Error(editErr))
					return nil, fmt.Errorf("failed to send ephemeral message: %w", editErr)
				}

				h.Logger.InfoContext(ctx, "Edit role response sent",
					attr.Any("result", result),
					attr.CorrelationIDFromMsg(msg))

				return nil, nil
			}

			if !resultPayload.Success {
				result, err := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"),
					fmt.Sprintf("Failed to update role: %s", resultPayload.Error))
				if err != nil {
					h.Logger.ErrorContext(ctx, "Failed to send ephemeral message", attr.Error(err))
					return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
				}

				h.Logger.InfoContext(ctx, "Edit role response sent",
					attr.Any("result", result),
					attr.CorrelationIDFromMsg(msg))

				return nil, nil
			}

			guildID := msg.Metadata.Get("guild_id")
			if guildID == "" {
				err := fmt.Errorf("guild ID missing from message metadata")
				h.Logger.ErrorContext(ctx, "Guild ID missing", attr.Error(err))

				result, editErr := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"),
					fmt.Sprintf("Failed to update role: %s", err))
				if editErr != nil {
					h.Logger.ErrorContext(ctx, "Failed to send ephemeral message", attr.Error(editErr))
					return nil, fmt.Errorf("failed to send ephemeral message: %w", editErr)
				}

				h.Logger.InfoContext(ctx, "Edit role response sent",
					attr.Any("result", result),
					attr.CorrelationIDFromMsg(msg))

				return nil, nil
			}

			// Add the role to the user in Discord
			roleResult, err := h.UserDiscord.GetRoleManager().AddRoleToUser(ctx, guildID, sharedtypes.DiscordID(discordID), discordRoleID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to add Discord role", attr.Error(err))

				result, editErr := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"),
					fmt.Sprintf("Role updated in application, but failed to sync with Discord: %s", err))
				if editErr != nil {
					h.Logger.ErrorContext(ctx, "Failed to send ephemeral message", attr.Error(editErr))
					return nil, fmt.Errorf("failed to send ephemeral message: %w", editErr)
				}

				h.Logger.InfoContext(ctx, "Edit role response sent",
					attr.Any("result", result),
					attr.CorrelationIDFromMsg(msg))

				return nil, nil
			}

			h.Logger.InfoContext(ctx, "Role added to user",
				attr.Any("role_result", roleResult),
				attr.CorrelationIDFromMsg(msg))

			// Send ephemeral message for successful update
			result, err := h.UserDiscord.GetRoleManager().EditRoleUpdateResponse(ctx, msg.Metadata.Get("correlation_id"), "Role update completed")
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send ephemeral message", attr.Error(err))
				return nil, fmt.Errorf("failed to send ephemeral message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Edit role response sent",
				attr.Any("result", result),
				attr.CorrelationIDFromMsg(msg))

			return nil, nil
		},
	)(msg)
}

// HandleAddRole handles the AddRole event (currently from the signup flow)
func (h *UserHandlers) HandleAddRole(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleAddRole",
		&discorduserevents.AddRolePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			rolePayload := payload.(*discorduserevents.AddRolePayload)

			// Attempt to add the role
			result, err := h.UserDiscord.GetRoleManager().AddRoleToUser(ctx, h.Config.GetGuildID(), rolePayload.UserID, rolePayload.RoleID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to add Discord role", attr.Error(err))
				// Create a failure message
				failurePayload := discorduserevents.RoleAdditionFailedPayload{
					UserID: rolePayload.UserID,
					Reason: err.Error(),
				}
				failureMsg, createErr := h.Helper.CreateResultMessage(msg, failurePayload, discorduserevents.SignupRoleAdditionFailed)
				if createErr != nil {
					h.Logger.ErrorContext(ctx, "Failed to create failure message", attr.Error(createErr))
					return nil, fmt.Errorf("failed to create failure message: %w", createErr)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Log the result details
			h.Logger.InfoContext(ctx, "Add role operation completed",
				attr.Any("result", result),
				attr.CorrelationIDFromMsg(msg))

			// If successful, create a success message
			successPayload := discorduserevents.RoleAddedPayload{
				UserID: rolePayload.UserID,
			}
			successMsg, createErr := h.Helper.CreateResultMessage(msg, successPayload, discorduserevents.SignupRoleAdded)
			if createErr != nil {
				h.Logger.ErrorContext(ctx, "Failed to create success message", attr.Error(createErr))
				return nil, fmt.Errorf("failed to create success message: %w", createErr)
			}

			return []*message.Message{successMsg}, nil
		},
	)(msg)
}
