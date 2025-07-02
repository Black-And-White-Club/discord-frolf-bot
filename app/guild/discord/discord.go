package discord

import (
	"context"
	"log/slog"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// GuildDiscordInterface defines the interface for GuildDiscord.
type GuildDiscordInterface interface {
	GetSetupManager() setup.SetupManager
	RegisterAllCommands(guildID string) error
	UnregisterAllCommands(guildID string) error
}

// GuildDiscord encapsulates all guild Discord services.
type GuildDiscord struct {
	session      discordgocommands.Session
	logger       *slog.Logger
	SetupManager setup.SetupManager
}

// NewGuildDiscord creates a new GuildDiscord instance.
func NewGuildDiscord(
	ctx context.Context,
	session discordgocommands.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (GuildDiscordInterface, error) {
	// Extract underlying session for setup manager (temporary workaround)
	underlyingSession := session.(*discordgocommands.DiscordSession).GetUnderlyingSession()
	setupManager, err := setup.NewSetupManager(underlyingSession, publisher, logger, helper, config, interactionStore, tracer, metrics)
	if err != nil {
		return nil, err
	}

	return &GuildDiscord{
		session:      session,
		logger:       logger,
		SetupManager: setupManager,
	}, nil
}

// GetSetupManager returns the SetupManager.
func (gd *GuildDiscord) GetSetupManager() setup.SetupManager {
	return gd.SetupManager
}

// RegisterAllCommands registers all bot commands for a guild after setup completion.
func (gd *GuildDiscord) RegisterAllCommands(guildID string) error {
	return discordgocommands.RegisterAllCommandsForGuild(gd.session, gd.logger, guildID)
}

// UnregisterAllCommands unregisters all bot commands from a guild during teardown.
func (gd *GuildDiscord) UnregisterAllCommands(guildID string) error {
	return discordgocommands.UnregisterAllCommandsForGuild(gd.session, gd.logger, guildID)
}
