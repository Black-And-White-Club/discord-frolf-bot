package discordhandlers

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillHandlers defines the interface for internal Watermill event handlers.
type Handlers interface {
	HandleSendDM(msg *message.Message) ([]*message.Message, error)
}

// UserHandlers handles user-related events.
type DiscordHandlers struct {
	Logger    observability.Logger
	Config    *config.Config
	EventUtil utils.EventUtil
	Helper    utils.Helpers
	Discord   discord.Operations
}

// NewDiscordHandlers creates a new DiscordHandlers struct.
func NewDiscordHandlers(
	logger observability.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	discord discord.Operations,
) Handlers {
	return &DiscordHandlers{
		Logger:    logger,
		Config:    config,
		EventUtil: eventUtil,
		Helper:    helper,
		Discord:   discord,
	}
}
