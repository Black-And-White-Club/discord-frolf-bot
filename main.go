package main

import (
	"context"
	"fmt"
	"net/http"
	pprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
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
	// Create context that will be cancelled on interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
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
	var cfg *config.Config
	var err error

	cfg, err = config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
		os.Exit(1)
	}

	obsConfig := observability.Config{
		ServiceName:     "discord-frolf-bot",
		Environment:     cfg.Observability.Environment,
		Version:         cfg.Service.Version,
		LokiURL:         cfg.Observability.LokiURL,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
		OTLPEndpoint:    cfg.Observability.OTLPEndpoint,
		OTLPTransport:   cfg.Observability.OTLPTransport,
		LogsEnabled:     cfg.Observability.OTLPLogsEnabled,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	logger := obs.Provider.Logger
	logger.Info("Observability initialized successfully")

	// --- Central Storage Hub Initialization ---
	// Initializing the shared storage container (Interaction Store + Guild Cache)
	appStores := storage.NewStores(ctx)

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
		discordgo.IntentGuildMembers |
		discordgo.IntentDirectMessages

	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// --- Bot Initialization ---
	discordBot, err := bot.NewDiscordBot(
		discordSessionWrapper,
		cfg,
		logger,
		appStores, // Pass the central storage hub instead of just an interaction store
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("discordbot"),
		utils.NewHelper(logger),
	)
	if err != nil {
		logger.Error("Failed to create Discord bot", attr.Error(err))
		os.Exit(1)
	}

	// --- Health Check Server ---
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"discord-frolf-bot","version":"%s"}`, cfg.Service.Version)
	})

	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if discordSession == nil || discordSession.State == nil || discordSession.State.User == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","reason":"discord_session_not_ready"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"discord-frolf-bot"}`)
	})

	if os.Getenv("PPROF_ENABLED") == "true" {
		addr := os.Getenv("PPROF_ADDR")
		if addr == "" {
			addr = ":6060"
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		go func() {
			logger.Info("pprof enabled", attr.String("addr", addr))
			if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
				logger.Error("pprof server failed", attr.Error(err))
			}
		}()
	}

	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = ":8080"
	}
	if !strings.HasPrefix(healthPort, ":") {
		healthPort = ":" + healthPort
	}

	healthServer := &http.Server{
		Addr:    healthPort,
		Handler: healthMux,
	}

	go func() {
		logger.Info("Starting health server", attr.String("addr", healthPort))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health server failed", attr.Error(err))
		}
	}()

	// --- Run Bot ---
	go func() {
		if err := discordBot.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Bot run failed", attr.Error(err))
		}
	}()

	logger.Info("Discord bot is running. Press Ctrl+C to gracefully shut down.")

	<-ctx.Done()
	logger.Info("Received shutdown signal, initiating graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := discordBot.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown Discord bot", attr.Error(err))
	}

	if err := obs.Provider.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown observability", attr.Error(err))
	}

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown health server", attr.Error(err))
	}

	logger.Info("Shutdown complete")
}

// runSetup handles the automated Discord server setup
func runSetup(ctx context.Context, guildID string) {
	fmt.Printf("Running setup for guild: %s\n", guildID)

	cfg, err := config.LoadBaseConfig()
	if err != nil {
		fmt.Printf("Failed to load base config: %v\n", err)
		os.Exit(1)
	}

	obsConfig := observability.Config{
		ServiceName: "discord-frolf-bot-setup",
		Environment: cfg.Observability.Environment,
		Version:     cfg.Service.Version,
		// ... other obs config remains the same
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	logger := obs.Provider.Logger

	// Initialize Storage Hub for setup context
	appStores := storage.NewStores(ctx)

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

	setupBot, err := bot.NewDiscordBot(
		discordSessionWrapper,
		cfg,
		logger,
		appStores, // Shared storage hub
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("setup"),
		utils.NewHelper(logger),
	)
	if err != nil {
		logger.Error("Failed to create setup bot", attr.Error(err))
		os.Exit(1)
	}

	if err := setupBot.Run(ctx); err != nil {
		logger.Error("Failed to start setup bot", attr.Error(err))
		os.Exit(1)
	}

	fmt.Println("Setup system initialized successfully!")
}

// runGatewayMode and runWorkerMode would follow a similar pattern once implemented...
func runGatewayMode(ctx context.Context) { /* ... */ }
func runWorkerMode(ctx context.Context)  { /* ... */ }
