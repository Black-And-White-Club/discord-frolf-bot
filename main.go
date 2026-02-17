package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	pprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	"golang.org/x/sync/errgroup"
)

func main() {
	// Create context that will be cancelled on interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Check for setup command line argument
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		guildID := ""
		if len(os.Args) >= 3 {
			guildID = os.Args[2]
		}
		if err := runSetup(ctx, guildID); err != nil {
			fmt.Fprintf(os.Stderr, "setup mode failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check bot mode for multi-pod deployment
	botMode := strings.TrimSpace(os.Getenv("BOT_MODE"))
	var runErr error
	switch botMode {
	case "gateway":
		runErr = runGatewayMode(ctx)
	case "worker":
		runErr = runWorkerMode(ctx)
	case "", "standalone":
		// Default: standalone mode (current behavior)
		runErr = runStandaloneMode(ctx)
	default:
		runErr = fmt.Errorf("unsupported BOT_MODE %q: supported values are standalone, gateway, worker", botMode)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "runtime failed: %v\n", runErr)
		os.Exit(1)
	}
}

// runStandaloneMode runs the bot in single-pod mode (current behavior)
func runStandaloneMode(ctx context.Context) error {
	var cfg *config.Config
	var err error

	cfg, err = config.LoadBaseConfig()
	if err != nil {
		return fmt.Errorf("failed to load base config: %w", err)
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
		return fmt.Errorf("failed to initialize observability: %w", err)
	}

	logger := obs.Provider.Logger
	logger.Info("Observability initialized successfully")

	// --- Central Storage Hub Initialization ---
	// Initializing the shared storage container (Interaction Store + Guild Cache)
	appStores := storage.NewStores(ctx)

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
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
		return fmt.Errorf("failed to create Discord bot: %w", err)
	}

	var runtimeErrMu sync.RWMutex
	var runtimeErr error
	setRuntimeErr := func(err error) {
		if err == nil {
			return
		}
		runtimeErrMu.Lock()
		if runtimeErr == nil {
			runtimeErr = err
		}
		runtimeErrMu.Unlock()
	}
	getRuntimeErr := func() error {
		runtimeErrMu.RLock()
		defer runtimeErrMu.RUnlock()
		return runtimeErr
	}

	// --- Health Check Server ---
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := getRuntimeErr(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","reason":"runtime_failed","service":"discord-frolf-bot","version":"%s"}`, cfg.Service.Version)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"discord-frolf-bot","version":"%s"}`, cfg.Service.Version)
	})

	healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := getRuntimeErr(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","reason":"runtime_failed"}`)
			return
		}
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
			addr = "127.0.0.1:6060"
		} else if strings.HasPrefix(addr, ":") {
			addr = "127.0.0.1" + addr
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		go func() {
			logger.Info("pprof enabled", attr.String("addr", addr))
			if !strings.HasPrefix(addr, "127.0.0.1:") && !strings.HasPrefix(addr, "localhost:") {
				logger.Warn("pprof is listening on a non-loopback address", attr.String("addr", addr))
			}
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

	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	defer runtimeCancel()
	eg, egCtx := errgroup.WithContext(runtimeCtx)

	eg.Go(func() error {
		logger.Info("Starting health server", attr.String("addr", healthPort))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			wrapped := fmt.Errorf("health server failed: %w", err)
			setRuntimeErr(wrapped)
			return wrapped
		}
		return nil
	})

	eg.Go(func() error {
		if err := discordBot.Run(egCtx); err != nil && !errors.Is(err, context.Canceled) {
			wrapped := fmt.Errorf("bot run failed: %w", err)
			setRuntimeErr(wrapped)
			return wrapped
		}
		return nil
	})

	logger.Info("Discord bot is running. Press Ctrl+C to gracefully shut down.")

	<-egCtx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		logger.Info("Received shutdown signal, initiating graceful shutdown...")
	} else {
		logger.Error("Runtime failed, initiating shutdown", attr.Error(getRuntimeErr()))
	}

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

	groupErr := eg.Wait()
	if groupErr != nil && !errors.Is(groupErr, context.Canceled) {
		setRuntimeErr(groupErr)
	}

	if err := getRuntimeErr(); err != nil {
		logger.Error("Shutdown completed after runtime failure", attr.Error(err))
		return err
	}

	logger.Info("Shutdown complete")
	return nil
}

// runSetup handles the automated Discord server setup
func runSetup(_ context.Context, guildID string) error {
	if guildID == "" {
		return fmt.Errorf("usage: go run main.go setup <guild_id>")
	}

	return fmt.Errorf("setup CLI mode is not supported in this build; run '/frolf-setup' in guild %s instead", guildID)
}

// runGatewayMode and runWorkerMode would follow a similar pattern once implemented...
func runGatewayMode(_ context.Context) error {
	return fmt.Errorf("BOT_MODE=gateway is not implemented; use standalone mode until gateway runtime is shipped")
}

func runWorkerMode(_ context.Context) error {
	return fmt.Errorf("BOT_MODE=worker is not implemented; use standalone mode until worker runtime is shipped")
}
