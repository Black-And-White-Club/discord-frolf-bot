package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleProcessRoundScoresFailed handles failed score processing events from the backend.
func (h *ScoreHandlers) HandleProcessRoundScoresFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper("HandleProcessRoundScoresFailed", &scoreevents.ProcessRoundScoresFailedPayloadV1{}, func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
		failurePayload, ok := payload.(*scoreevents.ProcessRoundScoresFailedPayloadV1)
		if !ok {
			return nil, fmt.Errorf("invalid payload type: expected *scoreevents.ProcessRoundScoresFailedPayloadV1")
		}

		h.Logger.ErrorContext(ctx, "Score processing failed",
			attr.String("round_id", failurePayload.RoundID.String()),
			attr.String("guild_id", string(failurePayload.GuildID)),
			attr.String("reason", failurePayload.Reason),
			attr.String("message_id", msg.UUID),
		)

		// TODO: Notify user in Discord about score processing failure
		// This should:
		// 1. Find the original interaction token from correlation metadata
		// 2. Send an ephemeral followup message to the user
		// 3. Include the specific error message from failurePayload.Reason
		// 4. Provide guidance on what to do next (retry, contact admin, etc.)

		return nil, nil
	})(msg)
}
