package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard"
	leaderboardrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/watermill"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/score"
	scorerouter "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq" // PostgreSQL driver
	"go.opentelemetry.io/otel/trace"
)

type DiscordBot struct {
	Session          discord.Session
	Logger           *slog.Logger
	Config           *config.Config
	EventBus         eventbus.EventBus
	InteractionStore storage.ISInterface
	Metrics          discordmetrics.DiscordMetrics
	Helper           utils.Helpers
	Tracer           trace.Tracer

	// Individual router instances per domain
	UserWatermillRouter        *message.Router
	RoundWatermillRouter       *message.Router
	ScoreWatermillRouter       *message.Router
	LeaderboardWatermillRouter *message.Router

	// Module routers
	UserRouter        *userrouter.UserRouter
	RoundRouter       *roundrouter.RoundRouter
	ScoreRouter       *scorerouter.ScoreRouter
	LeaderboardRouter *leaderboardrouter.LeaderboardRouter

	// Guild module reference for DB/config access
	GuildModule *guild.GuildModule
}

func NewDiscordBot(
	session discord.Session,
	cfg *config.Config,
	logger *slog.Logger,
	interactionStore storage.ISInterface,
	discordMetrics discordmetrics.DiscordMetrics,
	eventBusMetrics eventbusmetrics.EventBusMetrics,
	tracer trace.Tracer,
	helper utils.Helpers,
) (*DiscordBot, error) {
	logger.Info("Creating DiscordBot instance")

	eventBus, err := eventbus.NewEventBus(
		context.Background(),
		cfg.NATS.URL,
		logger,
		"discord",
		eventBusMetrics,
		tracer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
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

	return &DiscordBot{
		Session:                    session,
		Logger:                     logger,
		Config:                     cfg,
		EventBus:                   eventBus,
		InteractionStore:           interactionStore,
		Metrics:                    discordMetrics,
		Helper:                     helper,
		Tracer:                     tracer,
		UserWatermillRouter:        userRouter,
		RoundWatermillRouter:       roundRouter,
		ScoreWatermillRouter:       scoreRouter,
		LeaderboardWatermillRouter: leaderboardRouter,
	}, nil
}

func (bot *DiscordBot) Run(ctx context.Context) error {
	bot.Logger.Info("Starting Discord bot initialization")
	discordgoSession := bot.Session.(*discord.DiscordSession).GetUnderlyingSession()

	// Setup interaction registries
	registry := interactions.NewRegistry()
	reactionRegistry := interactions.NewReactionRegistry()
	reactionRegistry.RegisterWithSession(discordgoSession, bot.Session)

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
		bot.InteractionStore,
		bot.Metrics,
	)
	if err != nil {
		return fmt.Errorf("user module initialization failed: %w", err)
	}

	bot.RoundRouter, err = round.InitializeRoundModule(
		ctx,
		bot.Session,
		bot.RoundWatermillRouter,
		registry,
		reactionRegistry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.InteractionStore,
		bot.Metrics,
	)
	if err != nil {
		return fmt.Errorf("round module initialization failed: %w", err)
	}

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
		bot.InteractionStore,
		bot.Metrics,
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
		bot.InteractionStore,
		bot.Metrics,
	)
	if err != nil {
		return fmt.Errorf("leaderboard module initialization failed: %w", err)
	}
	// Initialize Guild Module
	// Try with database if available, fall back to no-database mode
	var db *sql.DB
	if dbURL := bot.Config.DatabaseURL; dbURL != "" {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err == nil && db.Ping() == nil {
			bot.Logger.Info("Guild module initializing with database support")
		} else {
			bot.Logger.Warn("Database connection failed, guild module will run without persistence", attr.Error(err))
			db = nil
		}
	} else {
		bot.Logger.Info("No database URL provided, guild module will run without persistence")
	}

	// Initialize Guild Module (with or without database)
	guildRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return fmt.Errorf("failed to create guild router: %w", err)
	}

	guildModule, err := guild.InitializeGuildModule(
		ctx,
		bot.Session,
		guildRouter,
		bot.EventBus,
		bot.EventBus,
		registry,
		bot.Logger,
		bot.Config,
		db, // Can be nil for no-database mode
	)
	if err != nil {
		return fmt.Errorf("guild module initialization failed: %w", err)
	}
	bot.GuildModule = guildModule

	// Start guild router
	go func() {
		if err := guildRouter.Run(ctx); err != nil && err != context.Canceled {
			bot.Logger.Error("Guild router failed", attr.Error(err))
		}
	}()

	if db != nil {
		defer db.Close()
	}

	// Only register commands per-guild in the GuildCreate handler. Do not register globally or on startup.

	// Register Discord handlers
	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.Logger.Info("Handling interaction", attr.String("type", i.Type.String()))
		registry.HandleInteraction(s, i)
	})

	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		bot.Logger.Info("Bot is ready", attr.Int("guilds", len(r.Guilds)))
	})

	// Register commands for every guild on join (instant per-guild registration)
	discordgoSession.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		bot.Logger.Info("Registering commands for guild", attr.String("guild_id", g.ID), attr.String("guild_name", g.Name))
		if err := discord.RegisterCommands(bot.Session, bot.Logger, g.ID); err != nil {
			bot.Logger.Error("Failed to register commands for guild", attr.Error(err), attr.String("guild_id", g.ID))
		}
		// Store guild in DB if not present
		go func() {
			ctx := context.Background()
			if bot.GuildModule != nil && bot.GuildModule.GetConfigHandler() != nil {
				err := bot.GuildModule.GetConfigHandler().EnsureGuildConfig(ctx, g.ID, g.Name)
				if err != nil {
					bot.Logger.Error("Failed to ensure guild config in DB", attr.Error(err), attr.String("guild_id", g.ID))
				}
			}
		}()
	})

	// Start the Watermill routers
	go func() {
		bot.Logger.Info("Starting User Watermill router")
		if err := bot.UserWatermillRouter.Run(ctx); err != nil {
			bot.Logger.Error("User Watermill router failed", attr.Error(err))
		}
	}()

	go func() {
		bot.Logger.Info("Starting Round Watermill router")
		if err := bot.RoundWatermillRouter.Run(ctx); err != nil {
			bot.Logger.Error("Round Watermill router failed", attr.Error(err))
		}
	}()

	go func() {
		bot.Logger.Info("Starting Score Watermill router")
		if err := bot.ScoreWatermillRouter.Run(ctx); err != nil {
			bot.Logger.Error("Score Watermill router failed", attr.Error(err))
		}
	}()

	go func() {
		bot.Logger.Info("Starting Leaderboard Watermill router")
		if err := bot.LeaderboardWatermillRouter.Run(ctx); err != nil {
			bot.Logger.Error("Leaderboard Watermill router failed", attr.Error(err))
		}
	}()

	// Open Discord connection
	if err := bot.Session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	bot.Logger.Info("Discord bot is now running")
	<-ctx.Done()
	return nil
}

func (bot *DiscordBot) Close() {
	bot.Logger.Info("Shutting down Discord bot...")

	// Close module routers first
	if bot.UserRouter != nil {
		bot.UserRouter.Close()
	}
	if bot.RoundRouter != nil {
		bot.RoundRouter.Close()
	}
	if bot.ScoreRouter != nil {
		bot.ScoreRouter.Close()
	}
	if bot.LeaderboardRouter != nil {
		bot.LeaderboardRouter.Close()
	}

	// Then close infrastructure
	if bot.UserWatermillRouter != nil {
		bot.UserWatermillRouter.Close()
	}
	if bot.RoundWatermillRouter != nil {
		bot.RoundWatermillRouter.Close()
	}
	if bot.ScoreWatermillRouter != nil {
		bot.ScoreWatermillRouter.Close()
	}
	if bot.LeaderboardWatermillRouter != nil {
		bot.LeaderboardWatermillRouter.Close()
	}
	if bot.Session != nil {
		bot.Session.Close()
	}
	if bot.EventBus != nil {
		bot.EventBus.Close()
	}

	bot.Logger.Info("Discord bot shutdown complete")
}

func (bot *DiscordBot) Shutdown(ctx context.Context) error {
	bot.Logger.Info("Shutting down Discord bot...")

	// Close module routers first
	if bot.UserRouter != nil {
		bot.UserRouter.Close()
	}
	if bot.RoundRouter != nil {
		bot.RoundRouter.Close()
	}
	if bot.ScoreRouter != nil {
		bot.ScoreRouter.Close()
	}
	if bot.LeaderboardRouter != nil {
		bot.LeaderboardRouter.Close()
	}

	// Then close infrastructure
	if bot.UserWatermillRouter != nil {
		bot.UserWatermillRouter.Close()
	}
	if bot.RoundWatermillRouter != nil {
		bot.RoundWatermillRouter.Close()
	}
	if bot.ScoreWatermillRouter != nil {
		bot.ScoreWatermillRouter.Close()
	}
	if bot.LeaderboardWatermillRouter != nil {
		bot.LeaderboardWatermillRouter.Close()
	}
	if bot.Session != nil {
		bot.Session.Close()
	}
	if bot.EventBus != nil {
		bot.EventBus.Close()
	}

	bot.Logger.Info("Discord bot shutdown complete")
	return nil
}
