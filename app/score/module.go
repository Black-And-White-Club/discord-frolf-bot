// app/score/module.go
package score

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel"
)

// InitializeUserModule initializes the user domain module.
func InitializeUserModule(
	ctx context.Context,
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	config *config.Config,
	helper utils.Helpers,
	discordMetricsService discordmetrics.DiscordMetrics,
) error {
	// Initialize Tracer
	tracer := otel.Tracer("score-module")

	// Initialize Watermill handlers (no need to register with router here)
	scorehandlers.NewScoreHandlers(logger, config, session, helper, tracer, discordMetricsService)

	return nil
}
