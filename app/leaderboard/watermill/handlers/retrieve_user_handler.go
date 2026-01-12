package leaderboardhandlers

import (
	"context"
	"fmt"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// HandleGetTagByDiscordID handles a request from Discord to get a user's tag.
func (h *LeaderboardHandlers) HandleGetTagByDiscordID(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling GetTagByDiscordID request")

	discordPayload := payload.(*sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1)

	userID := discordPayload.UserID

	// Correct backend payload
	backendPayload := leaderboardevents.SoloTagNumberRequestPayloadV1{
		GuildID: sharedtypes.GuildID(discordPayload.GuildID),
		UserID:  sharedtypes.DiscordID(userID),
	}

	h.Logger.InfoContext(ctx, "Successfully translated GetTagByDiscordID request",
		attr.String("user_id", string(userID)),
		attr.String("guild_id", discordPayload.GuildID))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.GetTagByUserIDRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleGetTagByDiscordIDResponse translates a backend tag number response to a Discord response.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDResponse(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling GetTagByDiscordIDResponse")

	backendPayload := payload.(*leaderboardevents.GetTagNumberResponsePayloadV1)

	var tagNumber sharedtypes.TagNumber
	if backendPayload.TagNumber != nil {
		tagNumber = *backendPayload.TagNumber
	}

	discordPayload := sharedleaderboardevents.LeaderboardTagAvailabilityResponsePayloadV1{
		TagNumber: tagNumber,
		GuildID:   string(backendPayload.GuildID),
		Available: backendPayload.Found,
	}

	h.Logger.InfoContext(ctx, "Successfully translated GetTagByDiscordIDResponse",
		attr.String("guild_id", string(backendPayload.GuildID)),
		attr.String("available", fmt.Sprintf("%v", backendPayload.Found)))

	return []handlerwrapper.Result{
		{
			Topic:   sharedleaderboardevents.LeaderboardTagAvailabilityResponseV1,
			Payload: discordPayload,
		},
	}, nil
}

// HandleGetTagByDiscordIDFailed handles a backend tag number lookup failure.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDFailed(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling GetTagByDiscordIDFailed")

	_ = payload.(*leaderboardevents.GetTagNumberFailedPayloadV1)

	// For now, just log the failure - Discord doesn't have a specific failure response for this
	h.Logger.WarnContext(ctx, "Tag number lookup failed on backend")

	return []handlerwrapper.Result{}, nil
}
