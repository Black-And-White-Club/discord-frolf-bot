package handlers

import (
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
)

// AuthHandlers handles auth-related Watermill events.
type AuthHandlers struct {
	logger           *slog.Logger
	cfg              *config.Config
	session          discord.Session
	interactionStore storage.ISInterface[any]
}

// NewAuthHandlers creates a new AuthHandlers instance.
func NewAuthHandlers(
	logger *slog.Logger,
	cfg *config.Config,
	session discord.Session,
	interactionStore storage.ISInterface[any],
) *AuthHandlers {
	return &AuthHandlers{
		logger:           logger,
		cfg:              cfg,
		session:          session,
		interactionStore: interactionStore,
	}
}
