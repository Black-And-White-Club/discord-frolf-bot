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

// HandleGetTagByDiscordID handles a request from Discord to get a user's tag.
func (h *LeaderboardHandlers) HandleGetTagByDiscordID(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGetTagByDiscordID",
		&sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling GetTagByDiscordID request", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1)

			userID := discordPayload.UserID

			// Correct backend payload
			backendPayload := leaderboardevents.SoloTagNumberRequestPayloadV1{
				GuildID: sharedtypes.GuildID(discordPayload.GuildID),
				UserID:  sharedtypes.DiscordID(userID),
			}

			// Correct event topic for backend to trigger
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.GetTagByUserIDRequestedV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
				return nil, fmt.Errorf("failed to create backend message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully translated GetTagByDiscordID request", attr.CorrelationIDFromMsg(msg))
			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleGetTagByDiscordIDResponse translates a backend tag response to a Discord response.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDResponse(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGetTagByDiscordIDResponse",
		nil,
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			topic := msg.Metadata.Get("topic")
			h.Logger.InfoContext(ctx, "Handling GetTagByDiscordIDResponse", attr.CorrelationIDFromMsg(msg), attr.Topic(topic))

			switch topic {
			case leaderboardevents.GetTagNumberResponseV1:
				var backendPayload leaderboardevents.GetTagNumberResponsePayloadV1
				if err := h.Helpers.UnmarshalPayload(msg, &backendPayload); err != nil {
					return nil, err
				}

				var tagNumber sharedtypes.TagNumber
				if backendPayload.TagNumber != nil {
					tagNumber = *backendPayload.TagNumber
				}

				discordPayload := sharedleaderboardevents.LeaderboardTagAvailabilityResponsePayloadV1{
					TagNumber: tagNumber,
					GuildID:   string(backendPayload.GuildID),
					ChannelID: msg.Metadata.Get("channel_id"),
					MessageID: msg.Metadata.Get("message_id"),
					Available: backendPayload.Found,
				}

				discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, sharedleaderboardevents.LeaderboardTagAvailabilityResponseV1)
				if err != nil {
					return nil, fmt.Errorf("failed to create discord message: %w", err)
				}
				return []*message.Message{discordMsg}, nil

			case leaderboardevents.GetTagNumberFailedV1:
				h.Logger.ErrorContext(ctx, "Received GetTagNumberFailed event")
				// Optionally handle the failure on the Discord side.
				return nil, nil

			default:
				return nil, fmt.Errorf("unexpected topic for tag number handler: %s", topic)
			}
		},
	)(msg)
}
