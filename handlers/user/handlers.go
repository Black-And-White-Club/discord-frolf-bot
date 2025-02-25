package userhandlers

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillHandlers defines the interface for internal Watermill event handlers.
type Handlers interface {
	HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateResult(msg *message.Message) ([]*message.Message, error)
	HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error)
	HandleUserCreated(msg *message.Message) ([]*message.Message, error)
	HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error)
	HandleSendUserDM(msg *message.Message) ([]*message.Message, error)
}

// UserHandlers handles user-related events.
type UserHandlers struct {
	Logger    observability.Logger
	Config    *config.Config
	EventUtil utils.EventUtil
	Helper    utils.Helpers
	Discord   discord.Operations
}

// NewUserHandlers creates a new UserHandlers struct.
func NewUserHandlers(
	logger observability.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	discord discord.Operations,
) Handlers {
	return &UserHandlers{
		Logger:    logger,
		Config:    config,
		EventUtil: eventUtil,
		Helper:    helper,
		Discord:   discord,
	}
}
