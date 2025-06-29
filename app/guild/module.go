package guild

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	guilddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	guildstorage "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/storage"
	guildrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill"
	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// GuildModule handles guild setup and configuration
type GuildModule struct {
	GuildDiscord  guilddiscord.GuildDiscordInterface
	configHandler *guildhandlers.GuildConfigHandler
	router        *guildrouter.GuildRouter
	logger        *slog.Logger
}

// InitializeGuildModule sets up the guild module with setup command and backend handlers
func InitializeGuildModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	publisher message.Publisher,
	subscriber message.Subscriber,
	interactionRegistry *interactions.Registry,
	logger *slog.Logger,
	cfg *config.Config,
	db *sql.DB,
) (*GuildModule, error) {
	// Create database service (or no-op if db is nil)
	var dbService guildhandlers.DatabaseService
	if db != nil {
		dbService = guildstorage.NewGuildDatabaseService(db, logger)
		logger.InfoContext(ctx, "Guild module using database persistence")
	} else {
		dbService = &NoOpDatabaseService{logger: logger}
		logger.InfoContext(ctx, "Guild module using no-op database service (no persistence)")
	}

	// Create GuildDiscord (Discord-specific managers)
	guildDiscord := guilddiscord.NewGuildDiscord(session, publisher, logger)

	// Create config handler for backend events
	configHandler := guildhandlers.NewGuildConfigHandler(logger, dbService)

	// Register setup command handler using the proper pattern
	setup.RegisterHandlers(interactionRegistry, guildDiscord.GetSetupManager())

	// Create guild router for watermill handlers
	guildRouter := guildrouter.NewGuildRouter(logger, router, subscriber, publisher)

	// Configure the router with handlers
	if err := guildRouter.Configure(ctx, configHandler); err != nil {
		logger.ErrorContext(ctx, "Failed to configure guild router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	module := &GuildModule{
		GuildDiscord:  guildDiscord,
		configHandler: configHandler,
		router:        guildRouter,
		logger:        logger,
	}

	logger.InfoContext(ctx, "Guild module initialized successfully")

	return module, nil
}

// NoOpDatabaseService is a no-op implementation for when no database is available
type NoOpDatabaseService struct {
	logger *slog.Logger
}

func (n *NoOpDatabaseService) SaveGuildConfig(ctx context.Context, config *guildstorage.GuildConfig) error {
	n.logger.InfoContext(ctx, "Guild config saved (no-op)",
		attr.String("guild_id", config.GuildID),
		attr.String("guild_name", config.GuildName))
	return nil
}

func (n *NoOpDatabaseService) GetGuildConfig(ctx context.Context, guildID string) (*guildstorage.GuildConfig, error) {
	return nil, fmt.Errorf("guild config not found (no database)")
}

func (n *NoOpDatabaseService) UpdateGuildConfig(ctx context.Context, guildID string, updates map[string]interface{}) error {
	n.logger.InfoContext(ctx, "Guild config updated (no-op)", attr.String("guild_id", guildID))
	return nil
}

// RegisterSetupCommand registers the /frolf-setup command with Discord
func RegisterSetupCommand(session discord.Session, logger *slog.Logger, guildID string) error {
	botUser, err := session.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}

	_, err = session.ApplicationCommandCreate(botUser.ID, guildID, &discordgo.ApplicationCommand{
		Name:        "frolf-setup",
		Description: "Set up Frolf Bot for this server (Admin only)",
	})
	if err != nil {
		logger.Error("Failed to create '/frolf-setup' command", attr.Error(err))
		return fmt.Errorf("failed to create '/frolf-setup' command: %w", err)
	}

	logger.Info("Registered command: /frolf-setup")
	return nil
}

// GetGuildDiscord returns the GuildDiscord interface for external use
func (m *GuildModule) GetGuildDiscord() guilddiscord.GuildDiscordInterface {
	return m.GuildDiscord
}

// GetConfigHandler returns the config handler for external use
func (m *GuildModule) GetConfigHandler() *guildhandlers.GuildConfigHandler {
	return m.configHandler
}
