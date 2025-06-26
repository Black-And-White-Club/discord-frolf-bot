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
	"github.com/bwmarrin/discordgo"
)

func main() {
	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Check for setup command line argument
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go setup <guild_id>")
			os.Exit(1)
		}
		guildID := os.Args[2]
		runSetup(ctx, guildID)
		return
	}

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
		Version:         cfg.Service.Version,
		LokiURL:         cfg.Observability.LokiURL,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	// Debug: Print the config to see what's being used
	fmt.Printf("DEBUG: Loki URL: %s\n", obsConfig.LokiURL)
	fmt.Printf("DEBUG: Service Name: %s\n", obsConfig.ServiceName)
	fmt.Printf("DEBUG: Environment: %s\n", obsConfig.Environment)

	// Initialize observability stack
	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	logger.Info("Observability initialized successfully")
	logger.Info("TEST LOG: Discord bot starting up")

	// --- Discord Components Initialization ---

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

	logger.Info("Discord bot initialized successfully")

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

	logger.Info("Discord bot is running. Press Ctrl+C to gracefully shut down.")

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
		} else {
			logger.Info("Discord bot shutdown successfully")
		}

		// Shutdown observability
		if err := obs.Provider.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown observability", attr.Error(err))
		} else {
			logger.Info("Observability shutdown successfully")
		}

		close(cleanShutdown)
	}()

	// Wait for shutdown to complete
	<-cleanShutdown
	logger.Info("Shutdown complete")
}

// runSetup handles the automated Discord server setup
func runSetup(ctx context.Context, guildID string) {
	fmt.Printf("Running setup for guild: %s\n", guildID)

	// Load config
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize basic observability for setup
	obsConfig := observability.Config{
		ServiceName: "discord-frolf-bot-setup",
		Environment: cfg.Observability.Environment,
		Version:     cfg.Service.Version,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		logger.Error("Failed to create Discord session", attr.Error(err))
		os.Exit(1)
	}

	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentGuildScheduledEvents |
		discordgo.IntentMessageContent |
		discordgo.IntentGuildMembers

	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// Create minimal bot for setup
	interactionStore := storage.NewInteractionStore()

	setupBot, err := bot.NewDiscordBot(
		discordSessionWrapper,
		cfg,
		logger,
		interactionStore,
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("setup"),
		utils.NewHelper(logger),
	)
	if err != nil {
		logger.Error("Failed to create setup bot", attr.Error(err))
		os.Exit(1)
	}

	// Open Discord session
	if err := setupBot.Session.Open(); err != nil {
		logger.Error("Failed to open Discord session", attr.Error(err))
		os.Exit(1)
	}
	defer setupBot.Session.Close()

	// Configure setup with better channel names and permissions
	setupConfig := bot.ServerSetupConfig{
		GuildID:              guildID,
		RequiredChannels:     []string{"signup", "events", "leaderboard"},
		RequiredRoles:        []string{"Rattler", "Editor", "Admin"},
		SignupEmojiName:      "🐍",
		CreateSignupMessage:  true,
		SignupMessageContent: "React with 🐍 to sign up for frolf events!",
		RegisteredRoleName:   "Rattler",
		AdminRoleName:        "Admin",
		ChannelPermissions: map[string]bot.ChannelPermissions{
			"signup": {
				RestrictPosting: false, // Handled specially - new users can see, players cannot
				AllowedRoles:    []string{},
			},
			"events": {
				RestrictPosting: true, // Only admins can post event embeds
				AllowedRoles:    []string{"Admin"},
			},
			"leaderboard": {
				RestrictPosting: true,
				AllowedRoles:    []string{"Admin"}, // Only admins can post leaderboard updates
			},
		},
	}

	// Run setup
	if err := setupBot.AutoSetupServer(ctx, setupConfig); err != nil {
		logger.Error("Setup failed", attr.Error(err))
		os.Exit(1)
	}

	fmt.Println("Setup completed successfully!")
	fmt.Println("Updated configuration:")
	fmt.Printf("  Guild ID: %s\n", setupBot.Config.Discord.GuildID)
	fmt.Printf("  Signup Channel ID: %s\n", setupBot.Config.Discord.SignupChannelID)
	fmt.Printf("  Event Channel ID: %s\n", setupBot.Config.Discord.EventChannelID)
	fmt.Printf("  Leaderboard Channel ID: %s\n", setupBot.Config.Discord.LeaderboardChannelID)
	fmt.Printf("  Signup Message ID: %s\n", setupBot.Config.Discord.SignupMessageID)
	fmt.Printf("  Registered Role ID: %s\n", setupBot.Config.Discord.RegisteredRoleID)
	fmt.Printf("  Admin Role ID: %s\n", setupBot.Config.Discord.AdminRoleID)
	fmt.Println("  Role Mappings:")
	for name, id := range setupBot.Config.Discord.RoleMappings {
		fmt.Printf("    %s: %s\n", name, id)
	}

	fmt.Println("\nNOTE: You need to manually update your config.yaml with these values.")
}
