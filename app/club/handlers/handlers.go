package handlers

import (
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/club/discord/challenge"
)

// ClubHandlers handles club-related Watermill events.
type ClubHandlers struct {
	logger           *slog.Logger
	challengeManager challenge.Manager
}

// NewClubHandlers creates a new ClubHandlers instance.
func NewClubHandlers(logger *slog.Logger, challengeManager challenge.Manager) Handlers {
	return &ClubHandlers{
		logger:           logger,
		challengeManager: challengeManager,
	}
}
