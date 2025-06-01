package leaderboardhandlers

import (
	"context"
	"fmt"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleTagAssignRequest translates a Discord tag assignment request to a backend request.
func (h *LeaderboardHandlers) HandleTagAssignRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignRequest",
		&discordleaderboardevents.LeaderboardTagAssignRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignRequest", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*discordleaderboardevents.LeaderboardTagAssignRequestPayload)

			targetUserID := sharedtypes.DiscordID(discordPayload.TargetUserID)
			requestorID := discordPayload.RequestorID
			tagNumber := sharedtypes.TagNumber(discordPayload.TagNumber)
			channelID := discordPayload.ChannelID
			messageID := discordPayload.MessageID

			if targetUserID == "" || requestorID == "" || tagNumber <= sharedtypes.TagNumber(0) || channelID == "" {
				err := fmt.Errorf("invalid TagAssignRequest payload: missing required fields")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg),
					attr.UserID(targetUserID),
					attr.UserID(requestorID),
					attr.Int("tag_number", int(tagNumber)),
				)
				return nil, err
			}

			// Validate and parse messageID as UUID for UpdateID
			if messageID == "" {
				err := fmt.Errorf("messageID is required but was empty")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg))
				return nil, err
			}

			updateID, err := uuid.Parse(messageID)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Invalid messageID format",
					attr.CorrelationIDFromMsg(msg),
					attr.String("messageID", messageID),
					attr.Error(err))
				return nil, fmt.Errorf("invalid messageID format '%s': %w", messageID, err)
			}

			backendPayload := leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     sharedtypes.DiscordID(targetUserID),
				TagNumber:  &tagNumber,
				UpdateID:   sharedtypes.RoundID(updateID),
				Source:     "manual",
				UpdateType: "new_tag",
			}

			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.LeaderboardTagAssignmentRequested)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create backend message: %w", err)
			}

			backendMsg.Metadata.Set("user_id", string(targetUserID))
			backendMsg.Metadata.Set("requestor_id", string(requestorID))
			backendMsg.Metadata.Set("channel_id", string(channelID))
			backendMsg.Metadata.Set("message_id", messageID)

			h.Logger.InfoContext(ctx, "Successfully translated TagAssignRequest", attr.CorrelationIDFromMsg(msg))
			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleTagAssignedResponse translates a backend TagAssigned event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignedResponse",
		&leaderboardevents.TagAssignedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.TagAssignedPayload)

			userID := msg.Metadata.Get("user_id")
			requestorID := msg.Metadata.Get("requestor_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			discordPayload := discordleaderboardevents.LeaderboardTagAssignedPayload{
				TargetUserID: string(backendPayload.UserID),
				TagNumber:    *backendPayload.TagNumber,
				ChannelID:    channelID,
				MessageID:    messageID,
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagAssignedTopic)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagAssignedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
				attr.String("requestor_id", requestorID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}

// HandleTagAssignFailedResponse translates a backend TagAssignmentFailed event to a Discord response.
func (h *LeaderboardHandlers) HandleTagAssignFailedResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagAssignFailedResponse",
		&leaderboardevents.TagAssignmentFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling TagAssignFailedResponse", attr.CorrelationIDFromMsg(msg))

			backendPayload := payload.(*leaderboardevents.TagAssignmentFailedPayload)

			userID := msg.Metadata.Get("user_id")
			requestorID := msg.Metadata.Get("requestor_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			discordPayload := discordleaderboardevents.LeaderboardTagAssignFailedPayload{
				TargetUserID: string(backendPayload.UserID),
				TagNumber:    *backendPayload.TagNumber,
				Reason:       backendPayload.Reason,
				ChannelID:    channelID,
				MessageID:    messageID,
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagAssignFailedTopic)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create discord message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated TagAssignFailedResponse",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", userID),
				attr.String("requestor_id", requestorID),
			)

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}
