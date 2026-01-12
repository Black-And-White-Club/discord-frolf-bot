package scorehandlers

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	sharedscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"go.opentelemetry.io/otel/trace"
)

// Handler defines the interface for score-related Watermill event handlers.
// Handlers is the typed interface used by the router. Methods are pure
// functions that accept a typed payload and return []handlerwrapper.Result.
type Handlers interface {
	HandleScoreUpdateRequestTyped(ctx context.Context, payload *sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateSuccessTyped(ctx context.Context, payload *scoreevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateFailureTyped(ctx context.Context, payload *scoreevents.ScoreUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleProcessRoundScoresFailedTyped(ctx context.Context, payload *scoreevents.ProcessRoundScoresFailedPayloadV1) ([]handlerwrapper.Result, error)
}

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	Logger         *slog.Logger
	Config         *config.Config
	Session        discord.Session
	Helper         utils.Helpers
	Tracer         trace.Tracer
	Metrics        discordmetrics.DiscordMetrics
}

// NewScoreHandlers creates a new ScoreHandlers struct.
func NewScoreHandlers(
	logger *slog.Logger,
	config *config.Config,
	session discord.Session,
	helpers utils.Helpers,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) *ScoreHandlers {
	return &ScoreHandlers{
		Logger:  logger,
		Config:  config,
		Session: session,
		Helper:  helpers,
		Tracer:  tracer,
		Metrics: metrics,
	}
}

