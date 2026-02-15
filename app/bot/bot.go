package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth"
	authrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/auth/watermill"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild"
	guildrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/router"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard"
	leaderboardrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/router"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/gateway"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/round/router"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/score"
	scorerouter "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/user/router"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
)

type DiscordBot struct {
	Session  discord.Session
	Logger   *slog.Logger
	Config   *config.Config
	EventBus eventbus.EventBus
	Storage  *storage.Stores
	Metrics  discordmetrics.DiscordMetrics
	Helper   utils.Helpers
	Tracer   trace.Tracer

	// Multi-tenant config resolver for per-guild config
	GuildConfigResolver guildconfig.GuildConfigResolver

	// Individual router instances per domain
	UserWatermillRouter        *message.Router
	RoundWatermillRouter       *message.Router
	ScoreWatermillRouter       *message.Router
	LeaderboardWatermillRouter *message.Router
	GuildWatermillRouter       *message.Router
	AuthWatermillRouter        *message.Router

	// Module routers
	UserRouter        *userrouter.UserRouter
	RoundRouter       *roundrouter.RoundRouter
	NativeEventMap    rounddiscord.NativeEventMap
	ScoreRouter       *scorerouter.ScoreRouter
	LeaderboardRouter *leaderboardrouter.LeaderboardRouter
	GuildRouter       *guildrouter.GuildRouter
	AuthRouter        *authrouter.AuthRouter

	// Shutdown synchronization
	shutdownOnce sync.Once

	// Command reconciliation (startup)
	commandRegistrar       func(discord.Session, *slog.Logger, string) error
	commandSyncDelay       time.Duration
	commandSyncWorkers     int
	commandSyncRetryDelay  time.Duration
	commandManifestVersion string
	commandSyncState       sync.Map // guildID -> manifest version
	commandSyncRetries     sync.Map // guildID -> struct{}

	gatewayStateMu       sync.RWMutex
	gatewaySessionID     string
	gatewayGuildCount    int
	gatewayEverConnected bool
}

const (
	defaultCommandSyncDelay   = 250 * time.Millisecond
	defaultCommandSyncWorkers = 4
	defaultCommandSyncRetry   = 5 * time.Second
	commandSyncDelayEnvVarKey = "DISCORD_COMMAND_SYNC_DELAY_MS"
	commandSyncWorkersEnvVar  = "DISCORD_COMMAND_SYNC_WORKERS"
	commandSyncRetryEnvVar    = "DISCORD_COMMAND_SYNC_RETRY_MS"
)

var (
	newEventBusFactory            = eventbus.NewEventBus
	newGuildConfigResolverFactory = guildconfig.NewResolver
)

func commandSyncDelayFromEnv(logger *slog.Logger) time.Duration {
	val := os.Getenv(commandSyncDelayEnvVarKey)
	if val == "" {
		return defaultCommandSyncDelay
	}
	ms, err := strconv.Atoi(val)
	if err != nil || ms < 0 {
		if logger != nil {
			logger.Warn("Invalid command sync delay; using default",
				attr.String("env", commandSyncDelayEnvVarKey),
				attr.String("value", val),
				attr.Duration("default", defaultCommandSyncDelay),
				attr.Error(err),
			)
		}
		return defaultCommandSyncDelay
	}
	return time.Duration(ms) * time.Millisecond
}

func commandSyncWorkersFromEnv(logger *slog.Logger) int {
	val := os.Getenv(commandSyncWorkersEnvVar)
	if val == "" {
		return defaultCommandSyncWorkers
	}
	workers, err := strconv.Atoi(val)
	if err != nil || workers <= 0 {
		if logger != nil {
			logger.Warn("Invalid command sync workers; using default",
				attr.String("env", commandSyncWorkersEnvVar),
				attr.String("value", val),
				attr.Int("default", defaultCommandSyncWorkers),
				attr.Error(err),
			)
		}
		return defaultCommandSyncWorkers
	}
	return workers
}

func commandSyncRetryDelayFromEnv(logger *slog.Logger) time.Duration {
	val := os.Getenv(commandSyncRetryEnvVar)
	if val == "" {
		return defaultCommandSyncRetry
	}

	ms, err := strconv.Atoi(val)
	if err != nil || ms <= 0 {
		if logger != nil {
			logger.Warn("Invalid command sync retry delay; using default",
				attr.String("env", commandSyncRetryEnvVar),
				attr.String("value", val),
				attr.Duration("default", defaultCommandSyncRetry),
				attr.Error(err),
			)
		}
		return defaultCommandSyncRetry
	}

	return time.Duration(ms) * time.Millisecond
}

func NewDiscordBot(
	session discord.Session,
	cfg *config.Config,
	logger *slog.Logger,
	appStores *storage.Stores, // This hub contains our new generic caches
	discordMetrics discordmetrics.DiscordMetrics,
	eventBusMetrics eventbusmetrics.EventBusMetrics,
	tracer trace.Tracer,
	helper utils.Helpers,
) (*DiscordBot, error) {
	if session == nil {
		return nil, fmt.Errorf("discord session is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if appStores == nil {
		return nil, fmt.Errorf("stores are required")
	}
	if appStores.GuildConfigCache == nil {
		return nil, fmt.Errorf("guild config cache store is required")
	}
	if appStores.InteractionStore == nil {
		return nil, fmt.Errorf("interaction store is required")
	}

	logger.Info("Creating DiscordBot instance")

	ctx := context.Background()

	eventBus, err := newEventBusFactory(
		ctx,
		cfg.NATS.URL,
		logger,
		"discord",
		eventBusMetrics,
		tracer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
	}

	// UPDATED: Pass the specific GuildConfigCache into the resolver.
	// This enables the "Short-Circuit" logic we just wrote in the resolver.
	guildConfigResolver, err := newGuildConfigResolverFactory(
		ctx,
		eventBus,
		appStores.GuildConfigCache,
		guildconfig.DefaultResolverConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize guild config resolver: %w", err)
	}

	// Create separate router instances for each domain
	userRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create user router: %w", err)
	}

	roundRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create round router: %w", err)
	}

	scoreRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create score router: %w", err)
	}

	leaderboardRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create leaderboard router: %w", err)
	}

	guildRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create guild router: %w", err)
	}

	authRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, fmt.Errorf("failed to create auth router: %w", err)
	}

	return &DiscordBot{
		Session:                    session,
		Logger:                     logger,
		Config:                     cfg,
		EventBus:                   eventBus,
		Storage:                    appStores, // InteractionStore is available here as ISInterface[any]
		Metrics:                    discordMetrics,
		Helper:                     helper,
		Tracer:                     tracer,
		GuildConfigResolver:        guildConfigResolver,
		UserWatermillRouter:        userRouter,
		RoundWatermillRouter:       roundRouter,
		ScoreWatermillRouter:       scoreRouter,
		LeaderboardWatermillRouter: leaderboardRouter,
		GuildWatermillRouter:       guildRouter,
		AuthWatermillRouter:        authRouter,
		commandRegistrar:           discord.RegisterCommands,
		commandSyncDelay:           commandSyncDelayFromEnv(logger),
		commandSyncWorkers:         commandSyncWorkersFromEnv(logger),
		commandSyncRetryDelay:      commandSyncRetryDelayFromEnv(logger),
		commandManifestVersion:     discord.GuildCommandManifestVersion(),
	}, nil
}

func (bot *DiscordBot) setGatewayContext(sessionID string, guildCount int) {
	bot.gatewayStateMu.Lock()
	bot.gatewaySessionID = sessionID
	bot.gatewayGuildCount = guildCount
	bot.gatewayStateMu.Unlock()
}

func (bot *DiscordBot) gatewayContext() (string, int) {
	bot.gatewayStateMu.RLock()
	defer bot.gatewayStateMu.RUnlock()
	return bot.gatewaySessionID, bot.gatewayGuildCount
}

func (bot *DiscordBot) markGatewayConnected() bool {
	bot.gatewayStateMu.Lock()
	defer bot.gatewayStateMu.Unlock()
	reconnect := bot.gatewayEverConnected
	bot.gatewayEverConnected = true
	return reconnect
}

func (bot *DiscordBot) registerGatewayLifecycleHandlers() {
	bot.Session.AddHandler(func(s *discordgo.Session, event *discordgo.Connect) {
		ctx := context.Background()
		reconnect := bot.markGatewayConnected()
		sessionID, guildCount := bot.gatewayContext()

		if bot.Metrics != nil {
			bot.Metrics.RecordWebsocketEvent(ctx, "connect")
			if reconnect {
				bot.Metrics.RecordWebsocketReconnect(ctx)
			}
		}

		bot.Logger.InfoContext(ctx, "Discord gateway connected",
			attr.Bool("reconnect", reconnect),
			attr.String("session_id", sessionID),
			attr.Int("guild_count", guildCount))
	})

	bot.Session.AddHandler(func(s *discordgo.Session, event *discordgo.Disconnect) {
		ctx := context.Background()
		sessionID, guildCount := bot.gatewayContext()

		if bot.Metrics != nil {
			bot.Metrics.RecordWebsocketDisconnect(ctx, "gateway_disconnect")
		}

		bot.Logger.WarnContext(ctx, "Discord gateway disconnected",
			attr.String("reason", "gateway_disconnect"),
			attr.String("session_id", sessionID),
			attr.Int("guild_count", guildCount))
	})

	bot.Session.AddHandler(func(s *discordgo.Session, event *discordgo.Resumed) {
		ctx := context.Background()
		sessionID, guildCount := bot.gatewayContext()

		if bot.Metrics != nil {
			bot.Metrics.RecordWebsocketEvent(ctx, "resumed")
			bot.Metrics.RecordWebsocketReconnect(ctx)
		}

		bot.Logger.InfoContext(ctx, "Discord gateway resumed session",
			attr.String("session_id", sessionID),
			attr.Int("guild_count", guildCount))
	})
}

func (bot *DiscordBot) Run(ctx context.Context) error {
	bot.Logger.Info("Starting Discord bot initialization")

	var commandSyncOnce sync.Once
	routerErrCh := make(chan error, 6)
	reportFatalRouterError := func(name string, err error) {
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		wrapped := fmt.Errorf("%s Watermill router failed: %w", name, err)
		bot.Logger.Error(wrapped.Error(), attr.Error(err))
		select {
		case routerErrCh <- wrapped:
		default:
		}
	}

	startRouter := func(name string, router *message.Router) {
		go func() {
			bot.Logger.Info(fmt.Sprintf("Starting %s Watermill router", name))
			if err := router.Run(ctx); err != nil {
				reportFatalRouterError(name, err)
			}
		}()
	}

	// Setup interaction registries
	registry := interactions.NewRegistry()
	registry.SetGuildConfigResolver(bot.GuildConfigResolver)
	registry.SetLogger(bot.Logger)

	// Initialize Reaction Registry with the bot's logger
	reactionRegistry := interactions.NewReactionRegistry(bot.Logger)
	reactionRegistry.RegisterWithSession(bot.Session, bot.Session)

	// Initialize Message Registry with the bot's logger
	messageRegistry := interactions.NewMessageRegistry(bot.Logger)
	messageRegistry.RegisterWithSession(bot.Session, bot.Session)

	// Initialize modules
	var err error
	bot.UserRouter, err = user.InitializeUserModule(
		ctx,
		bot.Session,
		bot.UserWatermillRouter,
		registry,
		reactionRegistry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.Storage.InteractionStore,
		bot.Storage.GuildConfigCache,
		bot.Metrics,
		bot.GuildConfigResolver,
	)
	if err != nil {
		return fmt.Errorf("user module initialization failed: %w", err)
	}

	roundResult, err := round.InitializeRoundModule(
		ctx,
		bot.Session,
		bot.RoundWatermillRouter,
		registry,
		reactionRegistry,
		messageRegistry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.Storage.InteractionStore,
		bot.Storage.GuildConfigCache,
		bot.Metrics,
		bot.GuildConfigResolver,
	)
	if err != nil {
		return fmt.Errorf("round module initialization failed: %w", err)
	}
	bot.RoundRouter = roundResult.Router
	bot.NativeEventMap = roundResult.NativeEventMap

	bot.ScoreRouter, err = score.InitializeScoreModule(
		ctx,
		bot.Session,
		bot.ScoreWatermillRouter,
		registry,
		reactionRegistry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
	)
	if err != nil {
		return fmt.Errorf("score module initialization failed: %w", err)
	}

	bot.LeaderboardRouter, err = leaderboard.InitializeLeaderboardModule(
		ctx,
		bot.Session,
		bot.LeaderboardWatermillRouter,
		registry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.GuildConfigResolver,
		bot.Storage.InteractionStore,
		bot.Storage.GuildConfigCache,
		bot.Metrics,
	)
	if err != nil {
		return fmt.Errorf("leaderboard module initialization failed: %w", err)
	}
	// Initialize Guild Module
	bot.GuildRouter, err = guild.InitializeGuildModule(
		ctx,
		bot.Session,
		bot.GuildWatermillRouter,
		registry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.Storage.InteractionStore,
		bot.Metrics,
		bot.GuildConfigResolver,
		bot.UserRouter.GetSignupManager(),
	)
	if err != nil {
		return fmt.Errorf("guild module initialization failed: %w", err)
	}

	// Initialize Auth Module
	bot.AuthRouter, err = auth.InitializeAuthModule(
		ctx,
		bot.Session,
		bot.AuthWatermillRouter,
		registry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.Storage.InteractionStore,
		bot.Metrics,
		bot.GuildConfigResolver,
	)
	if err != nil {
		return fmt.Errorf("auth module initialization failed: %w", err)
	}

	// Start guild router - MUST be running before Discord session opens
	// to ensure we can receive guild config retrieval responses
	startRouter("Guild", bot.GuildWatermillRouter)

	// Wait for guild router to be fully running before proceeding
	// This ensures the subscription for guild.config.retrieved.v1 is established
	// before we open Discord session and trigger config retrieval requests
	bot.Logger.Info("Waiting for Guild Watermill router to be running...")
	select {
	case <-ctx.Done():
		return nil
	case err := <-routerErrCh:
		return err
	case <-bot.GuildWatermillRouter.Running():
	}
	bot.Logger.Info("Guild Watermill router is now running")

	// Multi-tenant deployment: register all commands globally with proper gating
	// Setup command is admin-gated, other commands are setup-completion-gated
	bot.Logger.Info("Registering all commands globally for multi-tenant deployment")
	if err := discord.RegisterCommands(bot.Session, bot.Logger, ""); err != nil {
		return fmt.Errorf("failed to register global commands with Discord: %w", err)
	}

	// Register Discord handlers
	bot.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.Logger.Info("Handling interaction", attr.String("type", i.Type.String()))
		registry.HandleInteraction(s, i)
	})

	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		bot.setGatewayContext(r.SessionID, len(r.Guilds))
		if bot.Metrics != nil {
			bot.Metrics.RecordWebsocketEvent(ctx, "ready")
		}

		bot.Logger.Info("Bot is ready", attr.Int("guilds", len(r.Guilds)))
		for _, g := range r.Guilds {
			if g == nil || g.ID == "" {
				continue
			}
			if err := bot.requestGuildConfiguration(ctx, g.ID, g.Name); err != nil {
				bot.Logger.WarnContext(ctx, "Failed to warm guild configuration",
					attr.String("guild_id", g.ID),
					attr.Error(err))
			}
		}

		// Reconcile NativeEventMap from active Guild Scheduled Events (post-restart).
		// This ensures RSVP gateway listeners can resolve events immediately.
		// Note: Orphaned native event cleanup (CleanupOrphanedNativeEvents) is available
		// but not run on startup — the NativeEventMap is freshly populated and cannot
		// distinguish orphans without a backend round-existence check. Orphaned events
		// expire naturally via their ScheduledEndTime for V1.
		go gateway.ReconcileNativeEventMap(bot.Session, bot.NativeEventMap, r.Guilds, bot.Logger)

		commandSyncOnce.Do(func() {
			go bot.syncGuildCommands(ctx, r.Guilds)
		})
	})

	bot.registerGatewayLifecycleHandlers()

	// Handle guild lifecycle events for multi-tenant support
	bot.Session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		ctx := context.Background()

		bot.Logger.InfoContext(ctx, "Bot connected to guild",
			attr.String("guild_id", event.Guild.ID),
			attr.String("guild_name", event.Guild.Name),
			attr.Int("member_count", event.Guild.MemberCount))

		// Request guild configuration from backend
		// This will either return existing config or indicate that setup is needed
		if err := bot.requestGuildConfiguration(ctx, event.Guild.ID, event.Guild.Name); err != nil {
			bot.Logger.ErrorContext(ctx, "Failed to request guild configuration",
				attr.String("guild_id", event.Guild.ID),
				attr.Error(err))
		}
	})

	bot.Session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
		ctx := context.Background()

		bot.Logger.InfoContext(ctx, "Bot removed from guild",
			attr.String("guild_id", event.Guild.ID),
			attr.Bool("unavailable", event.Guild.Unavailable))

		// Only process actual removals, not temporary unavailability
		if event.Guild.Unavailable {
			bot.Logger.InfoContext(ctx, "Guild temporarily unavailable, not processing removal",
				attr.String("guild_id", event.Guild.ID))
			return
		}

		// Publish guild removal event to backend for cleanup
		if err := bot.publishGuildRemovedEvent(ctx, event.Guild.ID); err != nil {
			bot.Logger.ErrorContext(ctx, "Failed to publish guild removal event",
				attr.String("guild_id", event.Guild.ID),
				attr.Error(err))
		}
	})

	// Start the Watermill routers
	startRouter("User", bot.UserWatermillRouter)
	startRouter("Round", bot.RoundWatermillRouter)
	startRouter("Score", bot.ScoreWatermillRouter)
	startRouter("Leaderboard", bot.LeaderboardWatermillRouter)
	startRouter("Auth", bot.AuthWatermillRouter)

	// Open Discord connection
	if err := bot.Session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	bot.Logger.Info("Discord bot is now running")
	select {
	case <-ctx.Done():
		return nil
	case err := <-routerErrCh:
		return err
	}
}

func (bot *DiscordBot) syncGuildCommands(ctx context.Context, guilds []*discordgo.Guild) {
	if ctx == nil {
		ctx = context.Background()
	}

	if len(guilds) == 0 {
		bot.Logger.InfoContext(ctx, "No guilds in Ready payload; skipping command sync")
		return
	}

	bot.Logger.InfoContext(ctx, "Syncing guild commands for existing guilds",
		attr.Int("guilds", len(guilds)))

	registrar := bot.commandRegistrar
	if registrar == nil {
		registrar = discord.RegisterCommands
	}

	workers := bot.commandSyncWorkers
	if workers <= 0 {
		workers = 1
	}
	if workers > len(guilds) {
		workers = len(guilds)
	}
	if workers == 0 {
		return
	}

	var limiter <-chan time.Time
	var ticker *time.Ticker
	if bot.commandSyncDelay > 0 {
		ticker = time.NewTicker(bot.commandSyncDelay)
		limiter = ticker.C
		defer ticker.Stop()
	}

	waitForRateLimit := func(ctx context.Context) bool {
		if limiter == nil {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-limiter:
			return true
		}
	}

	guildJobs := make(chan *discordgo.Guild, len(guilds))
	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for g := range guildJobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if g == nil || g.ID == "" {
					continue
				}

				// Preserve the intended UX: only register guild commands after setup is complete.
				// This also avoids spamming unconfigured guilds with commands that will just be blocked.
				if bot.GuildConfigResolver != nil {
					if !bot.GuildConfigResolver.IsGuildSetupComplete(g.ID) {
						bot.Logger.InfoContext(ctx, "Skipping command sync: guild setup incomplete",
							attr.String("guild_id", g.ID))
						bot.scheduleGuildCommandRetry(ctx, g.ID)
						continue
					}
				}

				if bot.isGuildCommandManifestCurrent(g.ID) {
					bot.Logger.DebugContext(ctx, "Skipping command sync: manifest already current",
						attr.String("guild_id", g.ID),
						attr.String("manifest_version", bot.commandManifestVersion))
					continue
				}

				if !waitForRateLimit(ctx) {
					return
				}

				if err := registrar(bot.Session, bot.Logger, g.ID); err != nil {
					bot.Logger.ErrorContext(ctx, "Failed to sync guild commands",
						attr.String("guild_id", g.ID),
						attr.Error(err))
					continue
				}

				bot.markGuildCommandManifestSynced(g.ID)
			}
		})
	}

	for _, g := range guilds {
		select {
		case <-ctx.Done():
			close(guildJobs)
			wg.Wait()
			bot.Logger.InfoContext(ctx, "Command sync canceled", attr.Error(ctx.Err()))
			return
		case guildJobs <- g:
		}
	}
	close(guildJobs)
	wg.Wait()
}

func (bot *DiscordBot) scheduleGuildCommandRetry(ctx context.Context, guildID string) {
	if guildID == "" {
		return
	}
	if _, loaded := bot.commandSyncRetries.LoadOrStore(guildID, struct{}{}); loaded {
		return
	}

	retryDelay := bot.commandSyncRetryDelay
	if retryDelay <= 0 {
		retryDelay = defaultCommandSyncRetry
	}

	go func() {
		defer bot.commandSyncRetries.Delete(guildID)

		timer := time.NewTimer(retryDelay)
		defer timer.Stop()

		registrar := bot.commandRegistrar
		if registrar == nil {
			registrar = discord.RegisterCommands
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}

			if bot.isGuildCommandManifestCurrent(guildID) {
				return
			}

			if bot.GuildConfigResolver != nil && !bot.GuildConfigResolver.IsGuildSetupComplete(guildID) {
				timer.Reset(retryDelay)
				continue
			}

			if err := registrar(bot.Session, bot.Logger, guildID); err != nil {
				bot.Logger.ErrorContext(ctx, "Failed to sync guild commands on retry",
					attr.String("guild_id", guildID),
					attr.Error(err))
				timer.Reset(retryDelay)
				continue
			}

			bot.markGuildCommandManifestSynced(guildID)
			bot.Logger.InfoContext(ctx, "Synced guild commands after setup completion",
				attr.String("guild_id", guildID),
				attr.String("manifest_version", bot.commandManifestVersion))
			return
		}
	}()
}

func (bot *DiscordBot) isGuildCommandManifestCurrent(guildID string) bool {
	current, ok := bot.commandSyncState.Load(guildID)
	if !ok {
		return false
	}

	currentVersion, ok := current.(string)
	if !ok {
		return false
	}
	return currentVersion == bot.commandManifestVersion
}

func (bot *DiscordBot) markGuildCommandManifestSynced(guildID string) {
	if guildID == "" {
		return
	}
	bot.commandSyncState.Store(guildID, bot.commandManifestVersion)
}

func (bot *DiscordBot) Close() {
	// Just call Shutdown with background context for compatibility
	_ = bot.Shutdown(context.Background())
}

func (bot *DiscordBot) Shutdown(ctx context.Context) error {
	var shutdownErr error

	// Use sync.Once to ensure shutdown only happens once
	bot.shutdownOnce.Do(func() {
		bot.Logger.Info("Shutting down Discord bot...")

		// Close Discord session first to stop receiving events
		if bot.Session != nil {
			bot.Logger.Info("Closing Discord session...")
			if err := bot.Session.Close(); err != nil {
				bot.Logger.Warn("Error closing Discord session", attr.Error(err))
				shutdownErr = err
			}
			bot.Session = nil
		}

		// Close module routers (these handle business logic)
		if bot.UserRouter != nil {
			bot.Logger.Info("Closing user router...")
			if err := bot.UserRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing user router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.UserRouter = nil
		}
		if bot.RoundRouter != nil {
			bot.Logger.Info("Closing round router...")
			if err := bot.RoundRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing round router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.RoundRouter = nil
		}
		if bot.ScoreRouter != nil {
			bot.Logger.Info("Closing score router...")
			if err := bot.ScoreRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing score router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.ScoreRouter = nil
		}
		if bot.LeaderboardRouter != nil {
			bot.Logger.Info("Closing leaderboard router...")
			if err := bot.LeaderboardRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing leaderboard router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.LeaderboardRouter = nil
		}
		if bot.AuthRouter != nil {
			bot.Logger.Info("Closing auth router...")
			if err := bot.AuthRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing auth router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.AuthRouter = nil
		}

		// Close Watermill infrastructure routers
		if bot.UserWatermillRouter != nil {
			bot.Logger.Info("Closing user watermill router...")
			if err := bot.UserWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing user watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.UserWatermillRouter = nil
		}
		if bot.RoundWatermillRouter != nil {
			bot.Logger.Info("Closing round watermill router...")
			if err := bot.RoundWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing round watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.RoundWatermillRouter = nil
		}
		if bot.ScoreWatermillRouter != nil {
			bot.Logger.Info("Closing score watermill router...")
			if err := bot.ScoreWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing score watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.ScoreWatermillRouter = nil
		}
		if bot.LeaderboardWatermillRouter != nil {
			bot.Logger.Info("Closing leaderboard watermill router...")
			if err := bot.LeaderboardWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing leaderboard watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.LeaderboardWatermillRouter = nil
		}
		if bot.GuildWatermillRouter != nil {
			bot.Logger.Info("Closing guild watermill router...")
			if err := bot.GuildWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing guild watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.GuildWatermillRouter = nil
		}
		if bot.AuthWatermillRouter != nil {
			bot.Logger.Info("Closing auth watermill router...")
			if err := bot.AuthWatermillRouter.Close(); err != nil {
				bot.Logger.Warn("Error closing auth watermill router", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.AuthWatermillRouter = nil
		}

		// Close EventBus last (after all routers are closed)
		if bot.EventBus != nil {
			bot.Logger.Info("Closing event bus...")
			if err := bot.EventBus.Close(); err != nil {
				bot.Logger.Warn("Error closing event bus", attr.Error(err))
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
			bot.EventBus = nil
		}

		if shutdownErr != nil {
			bot.Logger.Error("Discord bot shutdown completed with errors", attr.Error(shutdownErr))
		} else {
			bot.Logger.Info("Discord bot shutdown complete")
		}
	})

	return shutdownErr
}

// requestGuildConfiguration requests configuration for a guild from the backend
func (bot *DiscordBot) requestGuildConfiguration(ctx context.Context, guildID, guildName string) error {
	if bot.GuildConfigResolver != nil {
		bot.GuildConfigResolver.RequestGuildConfigAsync(ctx, guildID)
		bot.Logger.InfoContext(ctx, "Requested guild config retrieval",
			attr.String("guild_id", guildID),
			attr.String("guild_name", guildName))
		return nil
	}

	// Create guild config retrieval request payload (best practice: only guild_id)
	payload := &guildevents.GuildConfigRetrievalRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(guildID),
	}

	// Create message for the correct event topic
	msg, err := bot.Helper.CreateNewMessage(payload, guildevents.GuildConfigRetrievalRequestedV1)
	if err != nil {
		return fmt.Errorf("failed to create guild config retrieval request message: %w", err)
	}

	msg.Metadata.Set("guild_id", guildID)

	// Publish guild config retrieval request event to backend
	if err := bot.EventBus.Publish(guildevents.GuildConfigRetrievalRequestedV1, msg); err != nil {
		return fmt.Errorf("failed to publish guild config retrieval request: %w", err)
	}

	bot.Logger.InfoContext(ctx, "Published guild config retrieval request to backend",
		attr.String("guild_id", guildID),
		attr.String("guild_name", guildName))

	return nil
}

// publishGuildRemovedEvent publishes a guild removal event to the backend
func (bot *DiscordBot) publishGuildRemovedEvent(ctx context.Context, guildID string) error {
	// Create guild removal payload
	payload := &guildevents.GuildConfigDeletionRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(guildID),
	}

	// Create message
	msg, err := bot.Helper.CreateNewMessage(payload, guildevents.GuildConfigDeletionRequestedV1)
	if err != nil {
		return fmt.Errorf("failed to create guild removal message: %w", err)
	}

	msg.Metadata.Set("guild_id", guildID)

	// Publish guild removal event to backend for cleanup
	if err := bot.EventBus.Publish(guildevents.GuildConfigDeletionRequestedV1, msg); err != nil {
		return fmt.Errorf("failed to publish guild removal event: %w", err)
	}

	bot.Logger.InfoContext(ctx, "Published guild removal event to backend",
		attr.String("guild_id", guildID))

	return nil
}
