package leaderboardhandlers

import (
	"context"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleLeaderboardRetrieveRequest handles a leaderboard retrieve request event from Discord.
func (h *LeaderboardHandlers) HandleLeaderboardRetrieveRequest(ctx context.Context,
	payload *discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard retrieve request")

	discordPayload := payload

	// Convert to backend payload
	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(discordPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully processed leaderboard retrieve request",
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
	payload *leaderboardevents.LeaderboardUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard updated notification")

	updatePayload := payload

	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: updatePayload.GuildID,
	}

	h.logger.InfoContext(ctx, "Requesting full leaderboard after update notification",
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
	payload *leaderboardevents.GetLeaderboardResponsePayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard response")

	payloadData := payload

	leaderboardData := make([]leaderboardtypes.LeaderboardEntry, len(payloadData.Leaderboard))
	for i, entry := range payloadData.Leaderboard {
		leaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			TagNumber: entry.TagNumber,
			UserID:    entry.UserID,
		}
	}

	discordPayload := discordleaderboardevents.LeaderboardRetrievedPayloadV1{
		Leaderboard: leaderboardData,
		GuildID:     string(payloadData.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully processed leaderboard data",
		attr.String("guild_id", string(payloadData.GuildID)),
		attr.Int("entry_count", len(leaderboardData)))

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardRetrievedV1,
			Payload: discordPayload,
		},
	}, nil
}
