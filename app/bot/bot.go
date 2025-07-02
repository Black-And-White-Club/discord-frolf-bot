package bot

import (
	"context"
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
	guildRouter, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return fmt.Errorf("failed to create guild router: %w", err)
	}

	_, err = guild.InitializeGuildModule(
		ctx,
		bot.Session,
		bot.EventBus,         // publisher
		bot.EventBus,         // subscriber
		guildRouter,          // messageRouter
		registry,             // interactionRegistry
		bot.Logger,           // logger
		bot.Helper,           // helper
		bot.Config,           // cfg
		bot.InteractionStore, // interactionStore
		bot.Tracer,           // tracer
		bot.Metrics,          // metrics
	)
	if err != nil {
		return fmt.Errorf("guild module initialization failed: %w", err)
	}

	// Start guild router
	go func() {
		if err := guildRouter.Run(ctx); err != nil && err != context.Canceled {
			bot.Logger.Error("Guild router failed", attr.Error(err))
		}
	}()

	// Multi-tenant deployment: register commands globally for all guilds
	bot.Logger.Info("Registering commands globally for multi-tenant deployment")
	if err := discord.RegisterCommands(bot.Session, bot.Logger, ""); err != nil {
		return fmt.Errorf("failed to register global commands with Discord: %w", err)
	}

	// Register Discord handlers
	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.Logger.Info("Handling interaction", attr.String("type", i.Type.String()))
		registry.HandleInteraction(s, i)
	})

	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		bot.Logger.Info("Bot is ready", attr.Int("guilds", len(r.Guilds)))
	})

	// Handle guild lifecycle events for multi-tenant support
	discordgoSession.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
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

	discordgoSession.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
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

// requestGuildConfiguration requests configuration for a guild from the backend
func (bot *DiscordBot) requestGuildConfiguration(ctx context.Context, guildID, guildName string) error {
	// TODO: Publish guild configuration request event to backend
	// The backend will respond with existing configuration or indicate setup is needed
	bot.Logger.InfoContext(ctx, "Requesting guild configuration from backend",
		attr.String("guild_id", guildID),
		attr.String("guild_name", guildName))

	// For now, just log that we would request configuration
	// In a real implementation, this would publish an event to the backend
	return nil
}

// publishGuildRemovedEvent publishes a guild removal event to the backend
func (bot *DiscordBot) publishGuildRemovedEvent(ctx context.Context, guildID string) error {
	// TODO: Publish guild removal event to backend for cleanup
	bot.Logger.InfoContext(ctx, "Publishing guild removal event to backend",
		attr.String("guild_id", guildID))

	// For now, just log that we would publish the event
	// In a real implementation, this would publish an event to the backend
	return nil
}
