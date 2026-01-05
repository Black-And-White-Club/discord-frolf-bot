package leaderboardhandlers

import (
	"context"
	"fmt"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Tag Swap --
// HandleTagSwapRequest translates a Discord tag swap request to a backend request.
func (h *LeaderboardHandlers) HandleTagSwapRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagSwapRequest",
		&sharedleaderboardevents.LeaderboardTagSwapRequestPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagSwapRequest", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*sharedleaderboardevents.LeaderboardTagSwapRequestPayloadV1)

			user1ID := sharedtypes.DiscordID(discordPayload.User1ID)
			user2ID := sharedtypes.DiscordID(discordPayload.User2ID)
			requestorID := sharedtypes.DiscordID(discordPayload.RequestorID)
			channelID := discordPayload.ChannelID
			messageID := discordPayload.MessageID

			if user1ID == "" || user2ID == "" || requestorID == "" || channelID == "" {
				err := fmt.Errorf("invalid TagSwapRequest payload: missing required fields")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg),
					attr.UserID(user1ID),
					attr.UserID(user2ID),
					attr.UserID(requestorID),
				)
				return nil, err
			}

			backendPayload := leaderboardevents.TagSwapRequestedPayloadV1{
				GuildID:     sharedtypes.GuildID(discordPayload.GuildID),
				RequestorID: requestorID,
				TargetID:    user2ID,
			}

			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.TagSwapRequestedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create backend message: %w", err)
			}

			backendMsg.Metadata.Set("user_id", string(requestorID))
			backendMsg.Metadata.Set("channel_id", channelID)
			backendMsg.Metadata.Set("message_id", messageID)

			h.Logger.InfoContext(ctx, "Successfully translated TagSwapRequest", attr.CorrelationIDFromMsg(msg))
			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleTagSwappedResponse translates a backend TagSwapProcessed event to a Discord response.
func (h *LeaderboardHandlers) HandleTagSwappedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagSwappedResponse",
		&leaderboardevents.TagSwapProcessedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagSwappedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.TagSwapProcessedPayloadV1)

			userID := msg.Metadata.Get("user_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			if userID == "" || channelID == "" {
				h.Logger.ErrorContext(ctx, "Missing required metadata for TagSwappedResponse", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing required metadata (user_id or channel_id)")
			}

			discordPayload := sharedleaderboardevents.LeaderboardTagSwappedPayloadV1{
				User1ID:   backendPayload.RequestorID,
				User2ID:   backendPayload.TargetID,
				ChannelID: channelID,
				MessageID: messageID,
				GuildID:   string(backendPayload.GuildID),
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, sharedleaderboardevents.LeaderboardTagSwappedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagSwappedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}

// HandleTagSwapFailedResponse translates a backend TagSwapFailed to a Discord response.
func (h *LeaderboardHandlers) HandleTagSwapFailedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagSwapFailedResponse",
		&leaderboardevents.TagSwapFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagSwapFailedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.TagSwapFailedPayloadV1)

			userID := msg.Metadata.Get("user_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			if userID == "" || channelID == "" {
				h.Logger.ErrorContext(ctx, "Missing required metadata for TagSwapFailedResponse", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing required metadata (user_id or channel_id)")
			}

			discordPayload := sharedleaderboardevents.LeaderboardTagSwapFailedPayloadV1{
				User1ID:   backendPayload.RequestorID,
				User2ID:   backendPayload.TargetID,
				Reason:    backendPayload.Reason,
				ChannelID: channelID,
				MessageID: messageID,
				GuildID:   string(backendPayload.GuildID),
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, sharedleaderboardevents.LeaderboardTagSwapFailedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagSwapFailedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}
