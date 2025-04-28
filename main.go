package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/bot"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func main() {
	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Configuration Loading ---
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// --- Observability Initialization ---
	obsConfig := observability.Config{
		ServiceName:     "discord-frolf-bot",
		Environment:     cfg.Observability.Environment,
		Version:         cfg.Service.Version, // Ensure version is from the service config
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	// Initialize observability stack
	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	// --- Discord Components Initialization ---

	// Initialize Watermill router
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		logger.Error("Failed to create Watermill router", attr.Error(err))
		os.Exit(1)
	}
	defer watermillRouter.Close()

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		logger.Error("Failed to create Discord session", attr.Error(err))
		os.Exit(1)
	}

	// Configure Discord intents
	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentGuildScheduledEvents |
		discordgo.IntentMessageContent |
		discordgo.IntentGuildMembers

	// Wrap Discord session with observability
	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// Create interaction store
	interactionStore := storage.NewInteractionStore()

	// --- Bot Initialization ---
	discordBot, err := bot.NewDiscordBot(
		discordSessionWrapper,
		cfg,
		logger,
		watermillRouter,
		interactionStore,
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("discordbot"),
		utils.NewHelper(logger),
	)
	if err != nil {
		logger.Error("Failed to create Discord bot", attr.Error(err))
		os.Exit(1)
	}

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{})

	// Start bot components
	go func() {
		logger.Info("Starting Discord bot components...")
		if err := discordBot.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Bot run failed", attr.Error(err))
			cancel()
		}
	}()

	// Shutdown handler
	go func() {
		select {
		case sig := <-interrupt:
			logger.Info("Received signal", attr.String("signal", sig.String()))
		case <-ctx.Done():
			logger.Info("Context cancelled")
		}

		logger.Info("Initiating graceful shutdown...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := discordBot.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown Discord bot", attr.Error(err))
		}
		close(cleanShutdown)
	}()

	// Wait for shutdown to complete
	<-cleanShutdown
	logger.Info("Shutdown complete")
}
