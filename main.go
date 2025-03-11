package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/bot"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill"
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logging
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize Loki logger
	logger, err := observability.NewLokiLogger(cfg.Loki.URL, cfg.Loki.Username, cfg.Loki.Password)
	if err != nil {
		log.Fatalf("Failed to initialize Loki logger: %v", err)
	}
	defer logger.Close()

	// Initialize OpenTelemetry/Tempo tracing
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

	// Create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize EventBus
	slogLogger.Info("Initializing EventBus...")
	eventBus, err := eventbus.NewEventBus(ctx, cfg.NATS.URL, slogLogger, "discord")
	if err != nil {
		log.Fatalf("Failed to create event bus: %v", err)
	}
	slogLogger.Info("EventBus initialized, setting up subscriptions...")
	defer eventBus.Close()

	// Initialize Watermill router (but don't start yet!)
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		log.Fatalf("Failed to create Watermill router: %v", err)
	}
	defer watermillRouter.Close()

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Set Discord intents
	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentGuildScheduledEvents |
		discordgo.IntentMessageContent |
		discordgo.IntentGuildMembers

	// Wrap the Discord session
	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// Create a single instance of InteractionStore
	interactionStore := storage.NewInteractionStore()

	// Create domain routers (ensure they register handlers before running the router)
	userDiscord, err := userdiscord.NewUserDiscord(ctx, discordSessionWrapper, eventBus, logger, utils.NewHelper(logger), cfg, interactionStore)
	if err != nil {
		log.Fatalf("Failed to create user Discord services: %v", err)
	}

	userRouter := userrouter.NewUserRouter(logger, watermillRouter, eventBus, eventBus, cfg, utils.NewHelper(logger), tracerInstance)
	slogLogger.Info("Configuring user router subscribers...")
	if err := userRouter.Configure(userhandlers.NewUserHandlers(logger, cfg, utils.NewEventUtil(), utils.NewHelper(logger), userDiscord), eventBus); err != nil {
		log.Fatalf("Failed to configure user router: %v", err)
	}
	slogLogger.Info("User  router subscribers configured.")

	roundDiscord, err := rounddiscord.NewRoundDiscord(ctx, discordSessionWrapper, eventBus, logger, utils.NewHelper(logger), cfg, interactionStore)
	if err != nil {
		log.Fatalf("Failed to create round Discord services: %v", err)
	}

	roundRouter := roundrouter.NewRoundRouter(logger, watermillRouter, eventBus, eventBus, cfg, utils.NewHelper(logger), tracerInstance)
	slogLogger.Info("Configuring round router subscribers...")
	if err := roundRouter.Configure(roundhandlers.NewRoundHandlers(logger, cfg, utils.NewEventUtil(), utils.NewHelper(logger), roundDiscord), eventBus); err != nil {
		log.Fatalf("Failed to configure round router: %v", err)
	}
	slogLogger.Info("Round router subscribers configured.")

	// ✅ Start Watermill Router AFTER Handlers Are Registered
	go func() {
		slogLogger.Info("Starting Watermill router...")
		if err := watermillRouter.Run(ctx); err != nil && err != context.Canceled {
			logger.Error(ctx, "Watermill router error", attr.Error(err))
		}
		slogLogger.Info("Watermill router stopped")
	}()

	// Create the Discord bot (but do NOT open the session yet)
	discordBot, err := bot.NewDiscordBot(discordSessionWrapper, cfg, logger, eventBus, watermillRouter, interactionStore)
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	// Start the bot in a goroutine AFTER Watermill is running
	go func() {
		slogLogger.Info("Starting Discord bot...")
		if err := discordBot.Run(ctx); err != nil && err != context.Canceled {
			logger.Error(ctx, "Discord bot error", attr.Error(err))
			cancel()
		}
		slogLogger.Info("Discord bot stopped")
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan // Block until a signal is received.
	logger.Info(context.Background(), "Shutting down gracefully...")
	cancel()

	// Close everything cleanly
	discordBot.Close()
	logger.Info(context.Background(), "Shutdown complete.")
}
