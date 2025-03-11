// app/user/watermill/handlers/user/handlers.go
package userhandlers

import (
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandler defines the interface for user-related Watermill event handlers.
type UserHandler interface {
	HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error)
	HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error)
	HandleUserCreated(msg *message.Message) ([]*message.Message, error)
	HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoleAdded(msg *message.Message) ([]*message.Message, error)
	HandleRoleAdditionFailed(msg *message.Message) ([]*message.Message, error)
	HandleAddRole(msg *message.Message) ([]*message.Message, error)
}

// UserHandlers handles user-related events.
type userHandlers struct {
	Logger      observability.Logger
	Config      *config.Config
	EventUtil   utils.EventUtil
	Helper      utils.Helpers
	UserDiscord userdiscord.UserDiscordInterface
}

// NewUserHandlers creates a new UserHandlers struct.
func NewUserHandlers(
	logger observability.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	userDiscord userdiscord.UserDiscordInterface,
) UserHandler {
	return &userHandlers{
		Logger:      logger,
		Config:      config,
		EventUtil:   eventUtil,
		Helper:      helper,
		UserDiscord: userDiscord,
	}
}
