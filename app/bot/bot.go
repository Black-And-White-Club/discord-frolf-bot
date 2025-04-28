package bot

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
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
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

type DiscordBot struct {
	Session          discord.Session
	Logger           *slog.Logger
	Config           *config.Config
	WatermillRouter  *message.Router
	EventBus         eventbus.EventBus
	InteractionStore storage.ISInterface
	Metrics          discordmetrics.DiscordMetrics
	Helper           utils.Helpers
	Tracer           trace.Tracer

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
	router *message.Router,
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

	return &DiscordBot{
		Session:          session,
		Logger:           logger,
		Config:           cfg,
		WatermillRouter:  router,
		EventBus:         eventBus,
		InteractionStore: interactionStore,
		Metrics:          discordMetrics,
		Helper:           helper,
		Tracer:           tracer,
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
		bot.WatermillRouter,
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
		bot.WatermillRouter,
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
		bot.WatermillRouter,
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
		bot.WatermillRouter,
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

	// Register Discord handlers
	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.Logger.Info("Handling interaction", attr.String("type", i.Type.String()))
		registry.HandleInteraction(s, i)
	})

	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		bot.Logger.Info("Bot is ready", attr.Int("guilds", len(r.Guilds)))
	})

	// Start the Watermill router
	go func() {
		bot.Logger.Info("Starting Watermill router")
		if err := bot.WatermillRouter.Run(ctx); err != nil {
			bot.Logger.Error("Watermill router failed", attr.Error(err))
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
	if bot.WatermillRouter != nil {
		bot.WatermillRouter.Close()
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
	if bot.WatermillRouter != nil {
		bot.WatermillRouter.Close()
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
