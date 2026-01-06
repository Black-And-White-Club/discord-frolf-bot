package leaderboardhandlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleLeaderboardUpdateFailed handles leaderboard update failure events.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleLeaderboardUpdateFailed",
		&leaderboardevents.LeaderboardUpdateFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*leaderboardevents.LeaderboardUpdateFailedPayloadV1)

			h.Logger.WarnContext(ctx, "Leaderboard update failed",
				attr.String("guild_id", string(p.GuildID)),
				attr.String("reason", p.Reason))

			// TODO: Notify admins/users via Discord that leaderboard update failed
			// This could be:
			// - Ephemeral message to the user who triggered the update
			// - Admin notification in a logging channel
			// - Update the leaderboard message with an error indicator

			return nil, nil
		},
	)(msg)
}

// HandleLeaderboardRetrievalFailed handles leaderboard retrieval failure events.
func (h *LeaderboardHandlers) HandleLeaderboardRetrievalFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleLeaderboardRetrievalFailed",
		&leaderboardevents.GetLeaderboardFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*leaderboardevents.GetLeaderboardFailedPayloadV1)

			h.Logger.WarnContext(ctx, "Leaderboard retrieval failed",
				attr.String("guild_id", string(p.GuildID)),
				attr.String("reason", p.Reason))

			// TODO: Notify requester that leaderboard couldn't be retrieved
			// This could be an ephemeral message to the user who requested it

			return nil, nil
		},
	)(msg)
}
