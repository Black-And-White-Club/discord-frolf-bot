package betting

import (
	"context"
	"fmt"
	"log/slog"

	bettingdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/betting/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/betting/discord/bet"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// InitializeBettingModule initializes the betting module components.
// Note: It doesn't return a Watermill Router since it has no NATS workers in discord-frolf-bot currently.
func InitializeBettingModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	logger *slog.Logger,
	cfg *config.Config,
	metrics discordmetrics.DiscordMetrics,
	tracer trace.Tracer,
) error {
	if tracer == nil {
		tracer = otel.Tracer("betting-module")
	}

	bettingDiscord, err := bettingdiscord.NewBettingDiscord(
		ctx,
		session,
		logger,
		cfg,
		tracer,
		metrics,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize betting Discord services", attr.Error(err))
		return fmt.Errorf("failed to initialize betting Discord services: %w", err)
	}

	// Register Discord interactions
	bet.RegisterHandlers(interactionRegistry, bettingDiscord.GetBetManager())

	return nil
}
