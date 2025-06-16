package round

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill"
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

// InitializeRoundModule initializes the Round domain module.
func InitializeRoundModule(
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
) (*roundrouter.RoundRouter, error) {
	tracer := otel.Tracer("round-module")

	// Initialize Discord services
	roundDiscord, err := rounddiscord.NewRoundDiscord(ctx, session, eventBus, logger, helper, cfg, interactionStore, tracer, discordMetrics)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize round Discord services", attr.Error(err))
		return nil, fmt.Errorf("failed to initialize round Discord services: %w", err)
	}

	// Register Discord interactions
	createround.RegisterHandlers(interactionRegistry, roundDiscord.GetCreateRoundManager())
	roundrsvp.RegisterHandlers(interactionRegistry, roundDiscord.GetRoundRsvpManager())
	deleteround.RegisterHandlers(interactionRegistry, roundDiscord.GetDeleteRoundManager())
	scoreround.RegisterHandlers(interactionRegistry, roundDiscord.GetScoreRoundManager())
	updateround.RegisterHandlers(interactionRegistry, roundDiscord.GetUpdateRoundManager()) // ADD THIS LINE

	// Build Watermill Handlers
	roundHandlers := roundhandlers.NewRoundHandlers(
		logger,
		cfg,
		helper,
		roundDiscord,
		tracer,
		discordMetrics,
	)

	// Setup Watermill router
	roundRouter := roundrouter.NewRoundRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	if err := roundRouter.Configure(ctx, roundHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure round router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	return roundRouter, nil
}
