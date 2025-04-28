package score

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	scorerouter "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill"
	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

// InitializeScoreModule initializes the score domain module.
func InitializeScoreModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	reactionRegistry *interactions.ReactionRegistry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
	discordMetrics discordmetrics.DiscordMetrics,
) (*scorerouter.ScoreRouter, error) {
	tracer := otel.Tracer("score-module")

	// Build Watermill Handlers first
	scoreHandlers := scorehandlers.NewScoreHandlers(
		logger,
		cfg,
		session,
		helper,
		tracer,
		discordMetrics,
	)

	// Setup Watermill router
	scoreRouter := scorerouter.NewScoreRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	// Configure with context and handlers
	if err := scoreRouter.Configure(ctx, scoreHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure score router",
			attr.Error(err),
		)
		return nil, fmt.Errorf("failed to configure score router: %w", err)
	}

	return scoreRouter, nil
}
