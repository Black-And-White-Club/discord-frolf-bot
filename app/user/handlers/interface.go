package handlers

import (
	"context"

	discorduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for user-related event handlers.
type Handlers interface {
	HandleUserCreated(ctx context.Context, payload *userevents.UserCreatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleUserCreationFailed(ctx context.Context, payload *userevents.UserCreationFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleAddRole(ctx context.Context, payload *discorduserevents.AddRolePayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleAdded(ctx context.Context, payload *discorduserevents.RoleAddedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleAdditionFailed(ctx context.Context, payload *discorduserevents.RoleAdditionFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateCommand(ctx context.Context, payload *discorduserevents.RoleUpdateCommandPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateButtonPress(ctx context.Context, payload *discorduserevents.RoleUpdateButtonPressPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdated(ctx context.Context, payload *userevents.UserRoleUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoleUpdateFailed(ctx context.Context, payload *userevents.UserRoleUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
}
