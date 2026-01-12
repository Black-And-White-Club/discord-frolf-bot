package leaderboardhandlers

import (
	"context"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// HandleLeaderboardRetrieveRequest handles a leaderboard retrieve request event from Discord.
func (h *LeaderboardHandlers) HandleLeaderboardRetrieveRequest(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling leaderboard retrieve request")

	discordPayload := payload.(*sharedleaderboardevents.LeaderboardRetrieveRequestPayloadV1)

	// Convert to backend payload
	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(discordPayload.GuildID),
	}

	h.Logger.InfoContext(ctx, "Successfully processed leaderboard retrieve request",
		attr.String("guild_id", discordPayload.GuildID))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.GetLeaderboardRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleLeaderboardUpdatedNotification handles backend.leaderboard.updated and re-requests full leaderboard.
func (h *LeaderboardHandlers) HandleLeaderboardUpdatedNotification(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling leaderboard updated notification")

	updatePayload := payload.(*leaderboardevents.LeaderboardUpdatedPayloadV1)

	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: updatePayload.GuildID,
	}

	h.Logger.InfoContext(ctx, "Requesting full leaderboard after update notification",
		attr.String("guild_id", string(updatePayload.GuildID)))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.GetLeaderboardRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleLeaderboardResponse handles backend.leaderboard.get.response and translates to Discord response.
func (h *LeaderboardHandlers) HandleLeaderboardResponse(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling leaderboard response")

	payloadData := payload.(*leaderboardevents.GetLeaderboardResponsePayloadV1)

	leaderboardData := make([]leaderboardtypes.LeaderboardEntry, len(payloadData.Leaderboard))
	for i, entry := range payloadData.Leaderboard {
		leaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			TagNumber: entry.TagNumber,
			UserID:    entry.UserID,
		}
	}

	discordPayload := sharedleaderboardevents.LeaderboardRetrievedPayloadV1{
		Leaderboard: leaderboardData,
		GuildID:     string(payloadData.GuildID),
	}

	h.Logger.InfoContext(ctx, "Successfully processed leaderboard data",
		attr.String("guild_id", string(payloadData.GuildID)),
		attr.Int("entry_count", len(leaderboardData)))

	return []handlerwrapper.Result{
		{
			Topic:   sharedleaderboardevents.LeaderboardRetrievedV1,
			Payload: discordPayload,
		},
	}, nil
}
