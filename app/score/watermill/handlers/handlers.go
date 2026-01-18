package scorehandlers

import (
	"context"
	"log/slog"

	discordscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handler defines the interface for score-related Watermill event handlers.
// Handlers is the typed interface used by the router. Methods are pure
// functions that accept a typed payload and return []handlerwrapper.Result.
type Handlers interface {
	HandleScoreUpdateRequestTyped(ctx context.Context, payload *discordscoreevents.ScoreUpdateRequestDiscordPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateSuccessTyped(ctx context.Context, payload *sharedevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateFailureTyped(ctx context.Context, payload *sharedevents.ScoreUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleProcessRoundScoresFailedTyped(ctx context.Context, payload *sharedevents.ProcessRoundScoresFailedPayloadV1) ([]handlerwrapper.Result, error)
}

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	logger *slog.Logger
}

// NewScoreHandlers creates a new ScoreHandlers struct.
func NewScoreHandlers(
	logger *slog.Logger,
) Handlers {
	return &ScoreHandlers{
		logger: logger,
	}
}
