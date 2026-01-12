package scorehandlers

import (
	"context"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleProcessRoundScoresFailed handles failed score processing events from the backend.
func (h *ScoreHandlers) HandleProcessRoundScoresFailedTyped(ctx context.Context, payload *scoreevents.ProcessRoundScoresFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, nil
	}

	h.Logger.ErrorContext(ctx, "Score processing failed",
		attr.String("round_id", payload.RoundID.String()),
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason),
	)

	// no downstream messages
	return nil, nil
}
