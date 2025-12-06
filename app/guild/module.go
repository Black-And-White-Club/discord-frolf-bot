package guild

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	guilddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	guildwatermill "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill"
	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"

	guildconfig "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
)

// InitializeGuildModule initializes the Guild domain module.
func InitializeGuildModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
	discordMetrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
	signupManager signup.SignupManager, // For tracking signup channels when configs are set up
) (*guildwatermill.GuildRouter, error) {
	tracer := otel.Tracer("guild-module")

	// Initialize Discord services
	guildDiscord, err := guilddiscord.NewGuildDiscord(
		ctx,
		session,
		eventBus,
		logger,
		helper,
		cfg,
		interactionStore,
		tracer,
		discordMetrics,
		guildConfigResolver,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize guild Discord services", attr.Error(err))
		return nil, fmt.Errorf("failed to initialize guild Discord services: %w", err)
	}

	// Register Discord interactions
	setup.RegisterHandlers(interactionRegistry, guildDiscord.GetSetupManager())

	// Build Watermill Handlers
	guildHandlers := guildhandlers.NewGuildHandlers(
		logger,
		cfg,
		helper,
		guildDiscord,
		guildConfigResolver,
		tracer,
		discordMetrics,
		signupManager,
	)

	// Setup Watermill router
	guildRouter := guildwatermill.NewGuildRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	if err := guildRouter.Configure(ctx, guildHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure guild router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	return guildRouter, nil
}
