package leaderboardhandlers

import (
	"context"
	"fmt"
	"log/slog"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// -- Tag Swap --
// HandleTagSwapRequest translates a Discord tag swap request to a backend request.
func (h *LeaderboardHandlers) HandleTagSwapRequest(ctx context.Context,
	payload *discordleaderboardevents.LeaderboardTagSwapRequestPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagSwapRequest")

	discordPayload := payload

	user1ID := sharedtypes.DiscordID(discordPayload.User1ID)
	user2ID := sharedtypes.DiscordID(discordPayload.User2ID)
	requestorID := sharedtypes.DiscordID(discordPayload.RequestorID)
	channelID := discordPayload.ChannelID
	messageID := discordPayload.MessageID

	if user1ID == "" || user2ID == "" || requestorID == "" || channelID == "" {
		err := fmt.Errorf("invalid TagSwapRequest payload: missing required fields")
		h.logger.ErrorContext(ctx, err.Error(),
			slog.Any("user1_id", user1ID),
			slog.Any("user2_id", user2ID),
			slog.Any("requestor_id", requestorID),
		)
		return nil, err
	}

	backendPayload := leaderboardevents.TagSwapRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(discordPayload.GuildID),
		RequestorID: requestorID,
		TargetID:    user2ID,
	}

	h.logger.InfoContext(ctx, "Successfully translated TagSwapRequest")
	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.TagSwapRequestedV1,
			Payload: backendPayload,
			Metadata: map[string]string{
				"user_id":    string(requestorID),
				"channel_id": channelID,
				"message_id": messageID,
			},
		},
	}, nil
}

// HandleTagSwappedResponse translates a backend TagSwapProcessed event to a Discord response.
func (h *LeaderboardHandlers) HandleTagSwappedResponse(ctx context.Context,
	payload *leaderboardevents.TagSwapProcessedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagSwappedResponse")

	backendPayload := payload

	discordPayload := discordleaderboardevents.LeaderboardTagSwappedPayloadV1{
		User1ID: backendPayload.RequestorID,
		User2ID: backendPayload.TargetID,
		GuildID: string(backendPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully translated TagSwappedResponse",
		slog.Any("user1_id", backendPayload.RequestorID),
		slog.Any("user2_id", backendPayload.TargetID),
	)

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardTagSwappedV1,
			Payload: discordPayload,
		},
	}, nil
}

// HandleTagSwapFailedResponse translates a backend TagSwapFailed to a Discord response.
func (h *LeaderboardHandlers) HandleTagSwapFailedResponse(ctx context.Context,
	payload *leaderboardevents.TagSwapFailedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling TagSwapFailedResponse")

	backendPayload := payload

	discordPayload := discordleaderboardevents.LeaderboardTagSwapFailedPayloadV1{
		User1ID: backendPayload.RequestorID,
		User2ID: backendPayload.TargetID,
		Reason:  backendPayload.Reason,
		GuildID: string(backendPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully translated TagSwapFailedResponse",
		slog.Any("user1_id", backendPayload.RequestorID),
		slog.Any("user2_id", backendPayload.TargetID),
		slog.String("reason", backendPayload.Reason),
	)

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardTagSwapFailedV1,
			Payload: discordPayload,
		},
	}, nil
}
