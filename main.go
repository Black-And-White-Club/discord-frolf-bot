package main

import (
	"context"
	"fmt"
	"net/http"
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

	// Check bot mode for multi-pod deployment
	botMode := os.Getenv("BOT_MODE")
	switch botMode {
	case "gateway":
		runGatewayMode(ctx)
	case "worker":
		runWorkerMode(ctx)
	default:
		// Default: standalone mode (current behavior)
		runStandaloneMode(ctx)
	}
}

// runStandaloneMode runs the bot in single-pod mode (current behavior)
func runStandaloneMode(ctx context.Context) {
	// --- Configuration Loading ---
	// Load base bot configuration (tokens, URLs, etc.) from environment/file
	// Discord server-specific configs will be loaded from backend via events
	var cfg *config.Config
	var err error

	// Load base configuration (no Discord server-specific settings)
	cfg, err = config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
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

	// --- Health Check Server ---
	// Start health check server for container orchestration
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"discord-frolf-bot","version":"%s"}`, cfg.Service.Version)
	})

	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check if Discord session is ready
		if discordSession == nil || discordSession.State == nil || discordSession.State.User == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","reason":"discord_session_not_ready"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"discord-frolf-bot"}`)
	})

	healthServer := &http.Server{
		Addr:    ":8080",
		Handler: healthMux,
	}

	go func() {
		logger.Info("Starting health check server on :8080")
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health server failed", attr.Error(err))
		}
	}()

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{})

	// Start bot components
	go func() {
		logger.Info("Starting Discord bot components...")
		if err := discordBot.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Bot run failed", attr.Error(err))
			// Note: we can't call cancel() here since it's not available in this scope
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

		// Shutdown health server
		if err := healthServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shutdown health server", attr.Error(err))
		} else {
			logger.Info("Health server shutdown successfully")
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

	// Load base config (no guild-specific settings)
	cfg, err := config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
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
		RequiredRoles:        []string{"User", "Editor", "Admin"},
		SignupEmojiName:      "🐍",
		CreateSignupMessage:  true,
		SignupMessageContent: "React with 🐍 to sign up for frolf events!",
		RegisteredRoleName:   "User",
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
	fmt.Println("Configuration details:")
	fmt.Printf("  Guild ID: %s\n", setupBot.Config.GetGuildID())
	fmt.Printf("  Signup Channel ID: %s\n", setupBot.Config.GetSignupChannelID())
	fmt.Printf("  Event Channel ID: %s\n", setupBot.Config.GetEventChannelID())
	fmt.Printf("  Leaderboard Channel ID: %s\n", setupBot.Config.GetLeaderboardChannelID())
	fmt.Printf("  Signup Message ID: %s\n", setupBot.Config.GetSignupMessageID())
	fmt.Printf("  Registered Role ID: %s\n", setupBot.Config.GetRegisteredRoleID())
	fmt.Printf("  Admin Role ID: %s\n", setupBot.Config.GetAdminRoleID())
	fmt.Println("  Role Mappings:")
	for name, id := range setupBot.Config.GetRoleMappings() {
		fmt.Printf("    %s: %s\n", name, id)
	}
	fmt.Println("\nConfiguration has been sent to the backend via events.")
	fmt.Println("The bot will automatically use this configuration when running.")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func init() {
	http.HandleFunc("/health", healthHandler)
}

// runGatewayMode runs only the Discord gateway connection (for multi-pod deployment)
func runGatewayMode(ctx context.Context) {
	fmt.Println("Starting Discord Gateway Mode - handling Discord connections only")

	// Load base configuration
	cfg, err := config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
		os.Exit(1)
	}

	// Initialize observability
	obsConfig := observability.Config{
		ServiceName:     "discord-frolf-bot-gateway",
		Environment:     cfg.Observability.Environment,
		Version:         cfg.Service.Version,
		LokiURL:         cfg.Observability.LokiURL,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	logger.Info("Gateway mode: Discord connection only, business logic handled by workers")

	// TODO: Implement gateway-only logic
	// - Create Discord session
	// - Register commands
	// - Handle Discord events
	// - Publish events to NATS (no local processing)

	fmt.Println("Gateway mode not yet implemented - falling back to standalone mode")
	runStandaloneMode(ctx)
}

// runWorkerMode runs only the business logic processing (for multi-pod deployment)
func runWorkerMode(ctx context.Context) {
	fmt.Println("Starting Worker Mode - handling business logic only")

	// Load base configuration
	cfg, err := config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
		os.Exit(1)
	}

	// Initialize observability
	obsConfig := observability.Config{
		ServiceName:     "discord-frolf-bot-worker",
		Environment:     cfg.Observability.Environment,
		Version:         cfg.Service.Version,
		LokiURL:         cfg.Observability.LokiURL,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	logger.Info("Worker mode: business logic only, no Discord connection")

	// TODO: Implement worker-only logic
	// - Subscribe to NATS events
	// - Process business logic
	// - Handle backend operations
	// - NO Discord session

	fmt.Println("Worker mode not yet implemented - falling back to standalone mode")
	runStandaloneMode(ctx)
}
