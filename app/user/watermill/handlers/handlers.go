// app/user/watermill/handlers/user/handlers.go
package userhandlers

import (
	"log/slog"

	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	service userdiscord.UserDiscordInterface
	helpers utils.Helpers
	config  *config.Config
	logger  *slog.Logger
}

// NewUserHandlers creates a new UserHandlers struct.
func NewUserHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	userDiscord userdiscord.UserDiscordInterface,
) Handlers {
	return &UserHandlers{
		service: userDiscord,
		helpers: helpers,
		config:  config,
		logger:  logger,
	}
}
