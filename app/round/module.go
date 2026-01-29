package round

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	scorecardupload "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/scorecard_upload"
	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/handlers"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/round/router"
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
	messageRegistry *interactions.MessageRegistry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	discordMetrics discordmetrics.DiscordMetrics,
	guildConfig guildconfig.GuildConfigResolver,
) (*roundrouter.RoundRouter, error) {
	tracer := otel.Tracer("round-module")

	// Initialize Discord services
	roundDiscord, err := rounddiscord.NewRoundDiscord(
		ctx,
		session,
		eventBus,
		logger,
		helper,
		cfg,
		interactionStore,
		guildConfigCache,
		guildConfig,
		tracer,
		discordMetrics,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize round discord", attr.Error(err))
		return nil, err
	}

	// Register interaction handlers
	createround.RegisterHandlers(interactionRegistry, roundDiscord.GetCreateRoundManager())
	roundrsvp.RegisterHandlers(interactionRegistry, roundDiscord.GetRoundRsvpManager())
	deleteround.RegisterHandlers(interactionRegistry, roundDiscord.GetDeleteRoundManager())
	scoreround.RegisterHandlers(interactionRegistry, roundDiscord.GetScoreRoundManager())
	updateround.RegisterHandlers(interactionRegistry, roundDiscord.GetUpdateRoundManager())
	scorecardupload.RegisterHandlers(interactionRegistry, messageRegistry, roundDiscord.GetScorecardUploadManager())

	// Build Watermill Handlers
	roundHandlers := handlers.NewRoundHandlers(
		logger,
		cfg,
		helper,
		roundDiscord,
		guildConfig,
	)

	// Setup Watermill router
	rr := roundrouter.NewRoundRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	if err := rr.Configure(ctx, roundHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure round router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	logger.InfoContext(ctx, "Round module initialized successfully")
	return rr, nil
}
