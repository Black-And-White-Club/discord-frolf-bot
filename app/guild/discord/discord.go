package guilddiscord

import (
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GuildDiscordInterface defines the interface for GuildDiscord services.
type GuildDiscordInterface interface {
	GetSetupManager() *setup.SetupManager
}

// GuildDiscord encapsulates all Discord-related managers/services for the guild domain.
type GuildDiscord struct {
	SetupManager *setup.SetupManager
}

// NewGuildDiscord creates a new GuildDiscord instance.
func NewGuildDiscord(
	discordSession discord.Session,
	publisher message.Publisher,
	logger *slog.Logger,
) GuildDiscordInterface {
	underlyingSession := discordSession.(*discord.DiscordSession).GetUnderlyingSession()
	setupManager := setup.NewSetupManager(underlyingSession, publisher, logger)
	return &GuildDiscord{
		SetupManager: setupManager,
	}
}

// GetSetupManager returns the SetupManager.
func (gd *GuildDiscord) GetSetupManager() *setup.SetupManager {
	return gd.SetupManager
}
