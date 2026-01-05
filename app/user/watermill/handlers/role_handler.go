package userhandlers

import (
	"context"
	"fmt"
	"strings"

	shareduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
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
		&shareduserevents.RoleUpdateCommandPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			cmdPayload := payload.(*shareduserevents.RoleUpdateCommandPayloadV1)

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
		&shareduserevents.RoleUpdateButtonPressPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			buttonPayload := payload.(*shareduserevents.RoleUpdateButtonPressPayloadV1)

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
			backendPayload := userevents.UserRoleUpdateRequestedPayloadV1{
				GuildID:     sharedtypes.GuildID(buttonPayload.GuildID),
				RequesterID: sharedtypes.DiscordID(buttonPayload.RequesterID),
				UserID:      sharedtypes.DiscordID(buttonPayload.TargetUserID),
				Role:        selectedRole,
			}
			backendEvent, err := h.Helper.CreateResultMessage(msg, backendPayload, userevents.UserRoleUpdateRequestedV1)
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

// HandleRoleUpdated processes the backend's success response.
func (h *UserHandlers) HandleRoleUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleUpdated",
		&userevents.UserRoleUpdatedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			resultPayload := payload.(*userevents.UserRoleUpdatedPayloadV1)

			h.Logger.InfoContext(ctx, "Role successfully updated",
				attr.UserID(resultPayload.UserID),
				attr.String("role", string(resultPayload.Role)),
				attr.CorrelationIDFromMsg(msg))

			// The backend has already updated the role, no further Discord action needed
			return nil, nil
		},
	)(msg)
}

// HandleRoleUpdateFailed processes the backend's failure response.
func (h *UserHandlers) HandleRoleUpdateFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoleUpdateFailed",
		&userevents.UserRoleUpdateFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			resultPayload := payload.(*userevents.UserRoleUpdateFailedPayloadV1)

			h.Logger.ErrorContext(ctx, "Role update failed",
				attr.UserID(resultPayload.UserID),
				attr.String("role", string(resultPayload.Role)),
				attr.String("reason", resultPayload.Reason),
				attr.CorrelationIDFromMsg(msg))

			// Log the failure - the backend has already handled the error
			return nil, nil
		},
	)(msg)
}

// HandleAddRole handles the AddRole event (currently from the signup flow)
func (h *UserHandlers) HandleAddRole(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleAddRole",
		&shareduserevents.AddRolePayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			rolePayload := payload.(*shareduserevents.AddRolePayloadV1)

			// Attempt to add the role
			result, err := h.UserDiscord.GetRoleManager().AddRoleToUser(ctx, h.Config.GetGuildID(), rolePayload.UserID, rolePayload.RoleID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to add Discord role", attr.Error(err))
				// Create a failure message
				failurePayload := shareduserevents.RoleAdditionFailedPayloadV1{
					UserID:  rolePayload.UserID,
					Reason:  err.Error(),
					GuildID: rolePayload.GuildID,
				}
				failureMsg, createErr := h.Helper.CreateResultMessage(msg, failurePayload, shareduserevents.SignupRoleAdditionFailedV1)
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
			successPayload := shareduserevents.RoleAddedPayloadV1{
				UserID:  rolePayload.UserID,
				GuildID: rolePayload.GuildID,
			}
			successMsg, createErr := h.Helper.CreateResultMessage(msg, successPayload, shareduserevents.SignupRoleAddedV1)
			if createErr != nil {
				h.Logger.ErrorContext(ctx, "Failed to create success message", attr.Error(createErr))
				return nil, fmt.Errorf("failed to create success message: %w", createErr)
			}

			return []*message.Message{successMsg}, nil
		},
	)(msg)
}
