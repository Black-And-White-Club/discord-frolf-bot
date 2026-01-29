package handlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleLeaderboardUpdateFailed handles leaderboard update failure events.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateFailed(ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdateFailedPayloadV1) ([]handlerwrapper.Result, error) {
	p := payload

	h.logger.WarnContext(ctx, "Leaderboard update failed",
		attr.String("guild_id", string(p.GuildID)),
		attr.String("reason", p.Reason))

	// TODO: Notify admins/users via Discord that leaderboard update failed
	// This could be:
	// - Ephemeral message to the user who triggered the update
	// - Admin notification in a logging channel
	// - Update the leaderboard message with an error indicator

	return []handlerwrapper.Result{}, nil
}

// HandleLeaderboardRetrievalFailed handles leaderboard retrieval failure events.
func (h *LeaderboardHandlers) HandleLeaderboardRetrievalFailed(ctx context.Context,
	payload *leaderboardevents.GetLeaderboardFailedPayloadV1) ([]handlerwrapper.Result, error) {
	p := payload

	h.logger.WarnContext(ctx, "Leaderboard retrieval failed",
		attr.String("guild_id", string(p.GuildID)),
		attr.String("reason", p.Reason))

	// TODO: Notify requester that leaderboard couldn't be retrieved
	// This could be an ephemeral message to the user who requested it

	return []handlerwrapper.Result{}, nil
}
