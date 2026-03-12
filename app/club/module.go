package club

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/club/discord/challenge"
	clubhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/club/handlers"
	clubrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/club/router"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

// InitializeClubModule initializes the club domain module.
func InitializeClubModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	guildConfigResolver guildconfig.GuildConfigResolver,
	discordMetrics discordmetrics.DiscordMetrics,
	createRoundManager createround.CreateRoundManager,
) (*clubrouter.ClubRouter, error) {
	tracer := otel.Tracer("club-module")

	challengeManager := challenge.NewManager(
		session,
		eventBus,
		logger,
		helper,
		cfg,
		guildConfigResolver,
		discordMetrics,
		createRoundManager,
	)
	challenge.RegisterHandlers(interactionRegistry, challengeManager)

	handlers := clubhandlers.NewClubHandlers(logger, challengeManager)
	clubRouter := clubrouter.NewClubRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	if err := clubRouter.Configure(ctx, handlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure club router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure club router: %w", err)
	}

	return clubRouter, nil
}
