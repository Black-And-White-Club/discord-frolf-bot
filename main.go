package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Black-And-White-Club/discord-frolf-bot/bot"
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

func main() {
	// Load configuration.
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger.
	logger, err := observability.NewLokiLogger(cfg.Loki.URL, cfg.Loki.Username, cfg.Loki.Password)
	if err != nil {
		log.Fatalf("Failed to initialize Loki logger: %v", err)
	}

	// Initialize OpenTelemetry/Tempo tracing.
	tracerInstance := &observability.TempoTracer{}
	tracingOpts := observability.TracingOptions{
		ServiceName:    cfg.Service.Name,
		TempoEndpoint:  cfg.Tempo.Endpoint,
		Insecure:       cfg.Tempo.Insecure,
		ServiceVersion: cfg.Tempo.ServiceVer,
		SampleRate:     cfg.Tempo.SampleRate,
	}

	tracerShutdown, err := tracerInstance.InitTracing(context.Background(), tracingOpts)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize tracing", attr.Error(err))
		return
	}
	defer tracerShutdown()

	// Initialize EventBus, use context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize a normal slog logger.
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	eventBus, err := eventbus.NewEventBus(ctx, cfg.NATS.URL, slogLogger) // Pass a regular logger
	if err != nil {
		log.Fatalf("Failed to create event bus: %v", err)
	}

	// Create Discord session.
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Set Discord intents.
	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentGuildScheduledEvents

	// Wrap the Discord session in the correct interface.
	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// Create an instance of the Operations interface.
	operations := discord.NewOperations(discordSessionWrapper, logger, cfg)

	// Create the Watermill message router.
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		log.Fatalf("Failed to create Watermill router: %v", err)
	}

	// Create the User Router.
	userRouter := userrouter.NewUserRouter(logger, watermillRouter, eventBus, eventBus, operations, cfg, utils.NewHelper(logger), tracerInstance)

	// Configure it after creation.
	if err := userRouter.Configure(userhandlers.NewUserHandlers(logger, cfg, utils.NewEventUtil(), utils.NewHelper(logger), operations), eventBus); err != nil {
		log.Fatalf("Failed to configure user router: %v", err)
	}

	// Create GatewayEventHandler.
	gatewayHandler := discord.NewGatewayEventHandler(eventBus, logger, utils.NewHelper(logger), cfg, discordSessionWrapper, operations)

	// Create the Discord bot, passing in dependencies.
	discordBot, err := bot.NewDiscordBot(discordSessionWrapper, cfg, gatewayHandler, logger, eventBus, watermillRouter, operations, *tracerInstance)
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	// Run the Discord bot in a goroutine.
	go func() {
		if err := discordBot.Run(ctx); err != nil && err != context.Canceled {
			logger.Error(ctx, "Discord bot error", attr.Error(err))
			cancel() // Stop the application if the bot fails.
		}
	}()

	// Handle graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan // Block until a signal is received.

	logger.Info(context.Background(), "Shutting down gracefully...")
	cancel() // Cancel the context, stopping the router and bot.

	// Close the Discord bot.
	discordBot.Close()

	// Close the EventBus (important for cleanup).
	if err := eventBus.Close(); err != nil {
		logger.Error(context.Background(), "Failed to close EventBus", attr.Error(err))
	}

	logger.Info(context.Background(), "Shutdown complete.")
}
