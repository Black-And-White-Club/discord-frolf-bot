package userhandlers

import (
	"context"
	"fmt"
	"strings"

	discorduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoleUpdateCommand handles the /rolerequest command.
func (h *UserHandlers) HandleRoleUpdateCommand(
	ctx context.Context,
	payload *discorduserevents.RoleUpdateCommandPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Respond to the role request with an ephemeral message
	// Note: interaction metadata would need to be passed in payload or context
	// For now, this is a placeholder that needs coordination with Discord module
	_ = h.service.GetRoleManager()
	return nil, nil
}

// HandleRoleUpdateButtonPress handles button interactions.
func (h *UserHandlers) HandleRoleUpdateButtonPress(
	ctx context.Context,
	payload *discorduserevents.RoleUpdateButtonPressPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Extract the selected role from the button's custom ID
	roleStr := strings.TrimPrefix(payload.InteractionCustomID, "role_button_")
	selectedRole := sharedtypes.UserRoleEnum(roleStr)

	// Acknowledge the button press with an ephemeral message
	_, err := h.service.GetRoleManager().RespondToRoleButtonPress(
		ctx,
		payload.InteractionID,
		payload.InteractionToken,
		sharedtypes.DiscordID(payload.RequesterID),
		string(selectedRole),
		sharedtypes.DiscordID(payload.TargetUserID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge interaction: %w", err)
	}

	// Create and publish a backend event for role update
	backendPayload := userevents.UserRoleUpdateRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(payload.GuildID),
		RequesterID: sharedtypes.DiscordID(payload.RequesterID),
		UserID:      sharedtypes.DiscordID(payload.TargetUserID),
		Role:        selectedRole,
	}

	return []handlerwrapper.Result{
		{Topic: userevents.UserRoleUpdateRequestedV1, Payload: &backendPayload},
	}, nil
}

// HandleRoleUpdated processes the backend's success response.
func (h *UserHandlers) HandleRoleUpdated(
	ctx context.Context,
	payload *userevents.UserRoleUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// The backend has already updated the role, no further Discord action needed
	return nil, nil
}

// HandleRoleUpdateFailed processes the backend's failure response.
func (h *UserHandlers) HandleRoleUpdateFailed(
	ctx context.Context,
	payload *userevents.UserRoleUpdateFailedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Log the failure - the backend has already handled the error
	return nil, nil
}

// HandleAddRole handles the AddRole event (currently from the signup flow)
func (h *UserHandlers) HandleAddRole(
	ctx context.Context,
	payload *discorduserevents.AddRolePayloadV1,
) ([]handlerwrapper.Result, error) {
	// Attempt to add the role
	_, err := h.service.GetRoleManager().AddRoleToUser(ctx, h.config.GetGuildID(), payload.UserID, payload.RoleID)
	if err != nil {
		// Create a failure result
		failurePayload := discorduserevents.RoleAdditionFailedPayloadV1{
			UserID:  payload.UserID,
			Reason:  err.Error(),
			GuildID: payload.GuildID,
		}
		return []handlerwrapper.Result{
			{Topic: discorduserevents.SignupRoleAdditionFailedV1, Payload: &failurePayload},
		}, nil
	}

	// If successful, create a success result
	successPayload := discorduserevents.RoleAddedPayloadV1{
		UserID:  payload.UserID,
		GuildID: payload.GuildID,
	}

	return []handlerwrapper.Result{
		{Topic: discorduserevents.SignupRoleAddedV1, Payload: &successPayload},
	}, nil
}
