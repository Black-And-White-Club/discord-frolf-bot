package discord

import (
	"context"
	"fmt"
	"log/slog"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/reset"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
)

// GuildDiscordInterface defines the interface for GuildDiscord.
type GuildDiscordInterface interface {
	GetSetupManager() setup.SetupManager
	GetResetManager() reset.ResetManager
	RegisterAllCommands(guildID string) error
	UnregisterAllCommands(guildID string) error
}

// GuildDiscord encapsulates all guild Discord services.
type GuildDiscord struct {
	session      discordgocommands.Session
	logger       *slog.Logger
	SetupManager setup.SetupManager
	ResetManager reset.ResetManager
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
	guildConfigResolver guildconfig.GuildConfigResolver,
) (GuildDiscordInterface, error) { // Use wrapped session directly; it implements required interface
	setupManager, err := setup.NewSetupManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics, guildConfigResolver)
	if err != nil {
		return nil, err
	}

	resetManager, err := reset.NewResetManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics)
	if err != nil {
		return nil, err
	}

	return &GuildDiscord{
		session:      session,
		logger:       logger,
		SetupManager: setupManager,
		ResetManager: resetManager,
	}, nil
}

// GetSetupManager returns the SetupManager.
func (gd *GuildDiscord) GetSetupManager() setup.SetupManager {
	return gd.SetupManager
}

// GetResetManager returns the ResetManager.
func (gd *GuildDiscord) GetResetManager() reset.ResetManager {
	return gd.ResetManager
}

// RegisterAllCommands registers all guild-specific commands for the given guild.
// This enables per-guild customization and ensures commands only appear after setup.
func (gd *GuildDiscord) RegisterAllCommands(guildID string) error {
	gd.logger.Info("Registering guild-specific commands after successful setup",
		attr.String("guild_id", guildID))

	// Use the discord commands package to register guild-specific commands
	return discordgocommands.RegisterCommands(gd.session, gd.logger, guildID)
}

// UnregisterAllCommands removes all guild-specific commands for the given guild.
// This is used when a guild is removed or configuration is reset.
func (gd *GuildDiscord) UnregisterAllCommands(guildID string) error {
	gd.logger.Info("Unregistering guild-specific commands",
		attr.String("guild_id", guildID))

	// Get bot user ID
	appID, err := gd.session.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}

	// Get all existing commands for this guild
	commands, err := gd.session.ApplicationCommands(appID.ID, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild commands: %w", err)
	}

	// Delete all guild-specific commands (except frolf-setup which is global)
	for _, cmd := range commands {
		if cmd.Name != "frolf-setup" {
			err = gd.session.ApplicationCommandDelete(appID.ID, guildID, cmd.ID)
			if err != nil {
				gd.logger.Error("Failed to delete guild command",
					attr.String("guild_id", guildID),
					attr.String("command_name", cmd.Name),
					attr.Error(err))
			} else {
				gd.logger.Info("Deleted guild command",
					attr.String("guild_id", guildID),
					attr.String("command_name", cmd.Name))
			}
		}
	}

	return nil
}
