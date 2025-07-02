package guild

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	guilddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	guildwatermill "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill"
	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// GuildModule handles guild setup and configuration
type GuildModule struct {
	discordModule   guilddiscord.GuildDiscordInterface
	responseHandler guildhandlers.Handlers
	router          *guildwatermill.GuildRouter
	logger          *slog.Logger
}

// InitializeGuildModule sets up the guild module with setup command and backend event handlers
func InitializeGuildModule(
	ctx context.Context,
	session discord.Session,
	publisher eventbus.EventBus,
	subscriber message.Subscriber,
	messageRouter *message.Router,
	interactionRegistry *interactions.Registry,
	logger *slog.Logger,
	helper utils.Helpers,
	cfg *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (*GuildModule, error) {
	// Create Discord module with all managers
	discordModule, err := guilddiscord.NewGuildDiscord(
		ctx,
		session, // Pass the Session interface directly
		publisher,
		logger,
		helper,
		cfg,
		interactionStore,
		tracer,
		metrics,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create guild discord module", attr.Error(err))
		return nil, fmt.Errorf("failed to create guild discord module: %w", err)
	}

	// Register setup command handler using the proper pattern
	setup.RegisterHandlers(interactionRegistry, discordModule.GetSetupManager())

	// Create response handler for backend events
	responseHandler := guildhandlers.NewGuildHandlers(
		logger,
		cfg,
		helper,
		discordModule,
		tracer,
		metrics,
	)

	// Create watermill router for backend event subscriptions
	guildRouter := guildwatermill.NewGuildRouter(
		logger,
		messageRouter,
		publisher,
		publisher,
		cfg,
		helper,
		tracer,
	)

	// Configure the router with response handlers
	if err := guildRouter.Configure(ctx, responseHandler); err != nil {
		logger.ErrorContext(ctx, "Failed to configure guild router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	module := &GuildModule{
		discordModule:   discordModule,
		responseHandler: responseHandler,
		router:          guildRouter,
		logger:          logger,
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
func (m *GuildModule) GetSetupManager() setup.SetupManager {
	return m.discordModule.GetSetupManager()
}

// GetResponseHandler returns the response handler for external use
func (m *GuildModule) GetResponseHandler() guildhandlers.Handlers {
	return m.responseHandler
}

// GetRouter returns the watermill router for external use
func (m *GuildModule) GetRouter() *guildwatermill.GuildRouter {
	return m.router
}

// Close gracefully shuts down the guild module
func (m *GuildModule) Close() error {
	if m.router != nil {
		return m.router.Close()
	}
	return nil
}
