package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/router/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session         discord.Session // Use the interface
	Logger          observability.Logger
	Config          *config.Config
	GatewayHandler  discord.GatewayEventHandler // Use the GatewayEventHandler
	watermillRouter *message.Router             // Add the main Watermill router
	eventbus        eventbus.EventBus           // Add the event bus

}

func NewDiscordBot(session discord.Session, cfg *config.Config, gatewayHandler discord.GatewayEventHandler, logger observability.Logger, eventBus eventbus.EventBus, router *message.Router, discord discord.Operations, tracer observability.TempoTracer) (*DiscordBot, error) {
	// Create the Watermill router *here*.
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}

	// Create domain routers, passing in the main router.
	userRouter := userrouter.NewUserRouter(logger, watermillRouter, eventBus, eventBus, discord, cfg, utils.NewHelper(logger), &tracer)

	// Call Configure AFTER initialization
	if err := userRouter.Configure(userhandlers.NewUserHandlers(logger, cfg, utils.NewEventUtil(), utils.NewHelper(logger), discord), eventBus); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}
	// ... add other domain routers ...
	bot := &DiscordBot{
		Session:         session,
		Logger:          logger,
		Config:          cfg,
		GatewayHandler:  gatewayHandler,
		watermillRouter: watermillRouter, // Store the router
		eventbus:        eventBus,
	}

	return bot, nil
}

func (bot *DiscordBot) Run(ctx context.Context) error {
	// Register the gateway event handlers.
	bot.GatewayHandler.RegisterHandlers()
	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		bot.Logger.Info(ctx, "Discord bot is connected and ready.")
	})
	// Open a websocket connection to Discord.
	err := bot.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening discord connection: %w", err)
	}

	bot.Logger.Info(ctx, "Discord bot is now running.")

	// Run the Watermill router (in a goroutine).
	go func() {
		if err := bot.watermillRouter.Run(ctx); err != nil && err != context.Canceled {
			bot.Logger.Error(ctx, "Watermill router error", attr.Error(err))
			// Consider how to handle router errors.  You might want to panic,
			// or send a signal to shut down the entire application.
		}
	}()
	// Wait for the main router to start running
	bot.Logger.Info(ctx, "Waiting for main router to start running")
	select {
	case <-bot.watermillRouter.Running():
		bot.Logger.Info(ctx, "Main router started and running")
	case <-time.After(time.Second * 5): // Increased timeout
		bot.Logger.Error(ctx, "Timeout waiting for main router to start")
		return fmt.Errorf("timeout waiting for main router to start")
	}
	// Block until the context is cancelled (e.g., by a signal).
	<-ctx.Done()
	bot.Logger.Info(ctx, "Shutting down Discord bot...")

	// Cleanly close the Discord session.
	if err := bot.Session.Close(); err != nil {
		bot.Logger.Error(ctx, "Failed to close discord session", attr.Error(err))
	}

	return nil
}

// Close closes the bot's resources
func (b *DiscordBot) Close() {
	b.Logger.Info(context.Background(), "Closing bot")
	// Close the Watermill router.
	if b.watermillRouter != nil {
		if err := b.watermillRouter.Close(); err != nil {
			b.Logger.Error(context.Background(), "Failed to close Watermill router", attr.Error(err))
		}
	}
}
