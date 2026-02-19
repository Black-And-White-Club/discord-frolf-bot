package leaderboard

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	claimtag "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/claim_tag" // Add this import
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/history"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/season"
	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/handlers"
	leaderboardrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/router"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

func InitializeLeaderboardModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	discordMetricsService discordmetrics.DiscordMetrics,
) (*leaderboardrouter.LeaderboardRouter, error) {
	// Initialize Tracer
	tracer := otel.Tracer("leaderboard-module")
	leaderboardDiscord, err := leaderboarddiscord.NewLeaderboardDiscord(
		ctx,
		session,
		publisher,
		logger,
		helper,
		cfg,
		guildConfigResolver,
		interactionStore,
		guildConfigCache,
		tracer,
		discordMetricsService,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize leaderboard Discord services", attr.Error(err))
		return nil, fmt.Errorf("failed to initialize leaderboard Discord services: %w", err)
	}

	// Register Discord interactions
	leaderboardupdated.RegisterHandlers(interactionRegistry, leaderboardDiscord.GetLeaderboardUpdateManager())
	claimtag.RegisterHandlers(interactionRegistry, leaderboardDiscord.GetClaimTagManager()) // Add this line
	season.RegisterHandlers(interactionRegistry, leaderboardDiscord.GetSeasonManager())
	history.RegisterHandlers(interactionRegistry, leaderboardDiscord.GetHistoryManager())

	// Initialize Watermill handlers
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(
		logger,
		cfg,
		helper,
		leaderboardDiscord,
		guildConfigResolver,
	)
	if leaderboardHandlers == nil {
		logger.ErrorContext(ctx, "Failed to create leaderboard handlers")
		return nil, fmt.Errorf("failed to create leaderboard handlers")
	}

	// Setup Watermill router
	leaderboardRouter := leaderboardrouter.NewLeaderboardRouter(
		logger,
		router,
		publisher,
		publisher,
		cfg,
		helper,
		tracer,
	)

	// Configure the router with context and handlers
	if err := leaderboardRouter.Configure(ctx, leaderboardHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure leaderboard router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure leaderboard router: %w", err)
	}

	return leaderboardRouter, nil
}
