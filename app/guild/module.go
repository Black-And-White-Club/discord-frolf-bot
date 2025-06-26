package guild

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
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
	setupManager  *setup.SetupManager
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
	// Create database service
	dbService := guildstorage.NewGuildDatabaseService(db, logger)

	// Create setup manager
	discordSession := session.(*discord.DiscordSession).GetUnderlyingSession()
	setupManager := setup.NewSetupManager(discordSession, publisher, logger)

	// Create config handler for backend events
	configHandler := guildhandlers.NewGuildConfigHandler(logger, dbService)

	// Register setup command handler using the proper pattern
	setup.RegisterHandlers(interactionRegistry, setupManager)

	// Create guild router for watermill handlers
	guildRouter := guildrouter.NewGuildRouter(logger, router, subscriber, publisher)

	// Configure the router with handlers
	if err := guildRouter.Configure(ctx, configHandler); err != nil {
		logger.ErrorContext(ctx, "Failed to configure guild router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	module := &GuildModule{
		setupManager:  setupManager,
		configHandler: configHandler,
		router:        guildRouter,
		logger:        logger,
	}

	logger.InfoContext(ctx, "Guild module initialized successfully")

	return module, nil
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

// GetSetupManager returns the setup manager for external use
func (m *GuildModule) GetSetupManager() *setup.SetupManager {
	return m.setupManager
}

// GetConfigHandler returns the config handler for external use
func (m *GuildModule) GetConfigHandler() *guildhandlers.GuildConfigHandler {
	return m.configHandler
}
