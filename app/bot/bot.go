package bot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild"
	guildrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill"
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

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
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

	// Multi-tenant config resolver for per-guild config
	GuildConfigResolver guildconfig.GuildConfigResolver

	// Individual router instances per domain
	UserWatermillRouter        *message.Router
	RoundWatermillRouter       *message.Router
	ScoreWatermillRouter       *message.Router
	LeaderboardWatermillRouter *message.Router
	GuildWatermillRouter       *message.Router

	// Module routers
	UserRouter        *userrouter.UserRouter
	RoundRouter       *roundrouter.RoundRouter
	ScoreRouter       *scorerouter.ScoreRouter
	LeaderboardRouter *leaderboardrouter.LeaderboardRouter
	GuildRouter       *guildrouter.GuildRouter

	// Shutdown synchronization
	shutdownOnce sync.Once
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

	// Create guild config resolver for multi-tenant configuration management
	guildConfigResolver := guildconfig.NewResolverWithDefaults(
		context.Background(),
		eventBus,
	)

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

	return &DiscordBot{
		Session:                    session,
		Logger:                     logger,
		Config:                     cfg,
		EventBus:                   eventBus,
		InteractionStore:           interactionStore,
		Metrics:                    discordMetrics,
		Helper:                     helper,
		Tracer:                     tracer,
		GuildConfigResolver:        guildConfigResolver,
		UserWatermillRouter:        userRouter,
		RoundWatermillRouter:       roundRouter,
		ScoreWatermillRouter:       scoreRouter,
		LeaderboardWatermillRouter: leaderboardRouter,
		GuildWatermillRouter:       guildRouter,
	}, nil
}

func (bot *DiscordBot) Run(ctx context.Context) error {
	bot.Logger.Info("Starting Discord bot initialization")
	discordgoSession := bot.Session.(*discord.DiscordSession).GetUnderlyingSession()

	// Setup interaction registries
	registry := interactions.NewRegistry()
	registry.SetGuildConfigResolver(bot.GuildConfigResolver)
	registry.SetLogger(bot.Logger)

	reactionRegistry := interactions.NewReactionRegistry()
	reactionRegistry.RegisterWithSession(discordgoSession, bot.Session)

	messageRegistry := interactions.NewMessageRegistry()
	messageRegistry.RegisterWithSession(discordgoSession, bot.Session)

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
		bot.GuildConfigResolver,
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
		messageRegistry,
		bot.EventBus,
		bot.Logger,
		bot.Config,
		bot.Helper,
		bot.InteractionStore,
		bot.Metrics,
		bot.GuildConfigResolver,
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
		bot.GuildConfigResolver,
		bot.InteractionStore,
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
		bot.InteractionStore,
		bot.Metrics,
		bot.GuildConfigResolver,
		bot.UserRouter.GetSignupManager(),
	)
	if err != nil {
		return fmt.Errorf("guild module initialization failed: %w", err)
	}

	// Start guild router
	go func() {
		bot.Logger.Info("Starting Guild Watermill router")
		if err := bot.GuildWatermillRouter.Run(ctx); err != nil && err != context.Canceled {
			bot.Logger.Error("Guild Watermill router failed", attr.Error(err))
		}
	}()

	// Multi-tenant deployment: register all commands globally with proper gating
	// Setup command is admin-gated, other commands are setup-completion-gated
	bot.Logger.Info("Registering all commands globally for multi-tenant deployment")
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
	// Create guild config retrieval request payload (best practice: only guild_id)
	payload := map[string]interface{}{
		"guild_id": guildID,
	}

	// Create message for the correct event topic
	msg, err := bot.Helper.CreateNewMessage(payload, "guild.config.retrieval_requested")
	if err != nil {
		return fmt.Errorf("failed to create guild config retrieval request message: %w", err)
	}

	msg.Metadata.Set("guild_id", guildID)

	// Publish guild config retrieval request event to backend
	if err := bot.EventBus.Publish("guild.config.retrieval_requested", msg); err != nil {
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
	payload := map[string]interface{}{
		"guild_id": guildID,
	}

	// Create message
	msg, err := bot.Helper.CreateNewMessage(payload, "guild.removed")
	if err != nil {
		return fmt.Errorf("failed to create guild removal message: %w", err)
	}

	msg.Metadata.Set("guild_id", guildID)

	// Publish guild removal event to backend for cleanup
	if err := bot.EventBus.Publish("guild.removed", msg); err != nil {
		return fmt.Errorf("failed to publish guild removal event: %w", err)
	}

	bot.Logger.InfoContext(ctx, "Published guild removal event to backend",
		attr.String("guild_id", guildID))

	return nil
}
