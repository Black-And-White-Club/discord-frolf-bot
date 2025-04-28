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
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel"
)

// InitializeRoundModule initializes the Round domain module.
func InitializeRoundModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	config *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
	discordMetricsService discordmetrics.DiscordMetrics,
) error {
	// Initialize Tracer
	tracer := otel.Tracer("round-module")

	// Initialize Discord services
	roundDiscord, err := rounddiscord.NewRoundDiscord(ctx, session, publisher, logger, helper, config, interactionStore, tracer, discordMetricsService)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize round Discord services", attr.Error(err))
		return err
	}

	// Register Discord interactions
	createround.RegisterHandlers(interactionRegistry, roundDiscord.GetCreateRoundManager())
	roundrsvp.RegisterHandlers(interactionRegistry, roundDiscord.GetRoundRsvpManager())
	deleteround.RegisterHandlers(interactionRegistry, roundDiscord.GetDeleteRoundManager())
	scoreround.RegisterHandlers(interactionRegistry, roundDiscord.GetScoreRoundManager())

	// Initialize Watermill handlers (no need to register with router here)
	roundHandlers := roundhandlers.NewRoundHandlers(logger, config, helper, roundDiscord, tracer, discordMetricsService)
	if roundHandlers == nil {
		logger.ErrorContext(ctx, "Failed to create round handlers")
		return fmt.Errorf("failed to create round handlers")
	}

	return nil
}
