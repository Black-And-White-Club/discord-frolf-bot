package userhandlers

import (
	"context"
	"fmt"

	discorduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUserCreated handles the UserCreated event from the backend.
func (h *UserHandlers) HandleUserCreated(
	ctx context.Context,
	payload *userevents.UserCreatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	rolePayload := discorduserevents.AddRolePayloadV1{
		UserID:  payload.UserID,
		RoleID:  h.config.GetRegisteredRoleID(),
		GuildID: string(payload.GuildID),
	}

	return []handlerwrapper.Result{
		{Topic: discorduserevents.SignupAddRoleV1, Payload: &rolePayload},
	}, nil
}

// HandleUserCreationFailed handles the UserCreationFailed event from the backend.
func (h *UserHandlers) HandleUserCreationFailed(
	ctx context.Context,
	payload *userevents.UserCreationFailedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Extract correlation ID from context if available (it should be in metadata)
	correlationID := ""
	if v := ctx.Value("correlation_id"); v != nil {
		correlationID = v.(string)
	}

	// Respond with the specific failure reason to the user
	_, err := h.userDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, false, payload.Reason)
	if err != nil {
		return nil, fmt.Errorf("failed to send signup failure: %w", err)
	}

	return nil, nil
}

// HandleRoleAdded handles the RoleAdded event.
func (h *UserHandlers) HandleRoleAdded(
	ctx context.Context,
	payload *discorduserevents.RoleAddedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Extract correlation ID from context if available
	correlationID := ""
	if v := ctx.Value("correlation_id"); v != nil {
		correlationID = v.(string)
	}

	_, err := h.userDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to send signup success: %w", err)
	}

	return nil, nil
}

// HandleRoleAdditionFailed handles the RoleAdditionFailed event.
func (h *UserHandlers) HandleRoleAdditionFailed(
	ctx context.Context,
	payload *discorduserevents.RoleAdditionFailedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Extract correlation ID from context if available
	correlationID := ""
	if v := ctx.Value("correlation_id"); v != nil {
		correlationID = v.(string)
	}

	_, err := h.userDiscord.GetSignupManager().SendSignupResult(ctx, correlationID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to send signup failure: %w", err)
	}

	return nil, nil
}
