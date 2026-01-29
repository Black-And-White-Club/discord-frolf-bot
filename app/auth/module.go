package auth

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	authdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/permission"
	authwatermill "github.com/Black-And-White-Club/discord-frolf-bot/app/auth/watermill"
	authhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/auth/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

// InitializeAuthModule initializes the auth module for PWA dashboard access.
func InitializeAuthModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface[any],
	discordMetrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) (*authwatermill.AuthRouter, error) {
	tracer := otel.Tracer("auth-module")

	// Initialize permission mapper
	permMapper := permission.NewMapper()

	// Initialize Auth Discord services
	authDiscord, err := authdiscord.NewAuthDiscord(
		ctx,
		session,
		eventBus,
		logger,
		cfg,
		tracer,
		discordMetrics,
		guildConfigResolver,
		permMapper,
		interactionStore,
		helper,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize Auth Discord services", attr.Error(err))
		return nil, fmt.Errorf("failed to initialize Auth Discord services: %w", err)
	}

	// Register slash command handlers
	dashboard.RegisterHandlers(interactionRegistry, authDiscord.GetDashboardManager())

	// Build Watermill handlers
	authHandlers := authhandlers.NewAuthHandlers(
		logger,
		cfg,
		session,
		interactionStore,
	)

	// Setup Watermill router
	authRouter := authwatermill.NewAuthRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)

	if err := authRouter.Configure(ctx, authHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure Auth router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure Auth router: %w", err)
	}

	return authRouter, nil
}
