package leaderboardhandlers

import (
	"context"
	"fmt"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetTagByDiscordID handles a request from Discord to get a user's tag.
func (h *LeaderboardHandlers) HandleGetTagByDiscordID(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGetTagByDiscordID",
		&discordleaderboardevents.LeaderboardTagAvailabilityRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling GetTagByDiscordID request", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*discordleaderboardevents.LeaderboardTagAvailabilityRequestPayload)

			userID := discordPayload.UserID

			// Correct backend payload
			backendPayload := leaderboardevents.SoloTagNumberRequestPayload{
				UserID: sharedtypes.DiscordID(userID),
			}

			// Correct event topic for backend to trigger
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.GetTagByUserIDRequest)
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
			case leaderboardevents.GetTagNumberResponse:
				var backendPayload leaderboardevents.GetTagNumberResponsePayload
				if err := h.Helpers.UnmarshalPayload(msg, &backendPayload); err != nil {
					return nil, err
				}

				discordPayload := discordleaderboardevents.LeaderboardTagAvailabilityResponsePayload{
					TagNumber: *backendPayload.TagNumber,
				}

				discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, leaderboardevents.GetTagByUserIDResponse)
				if err != nil {
					return nil, fmt.Errorf("failed to create discord message: %w", err)
				}
				return []*message.Message{discordMsg}, nil

			case leaderboardevents.GetTagNumberFailed:
				h.Logger.ErrorContext(ctx, "Received GetTagNumberFailed event")
				// Optionally handle the failure on the Discord side.
				return nil, nil

			default:
				return nil, fmt.Errorf("unexpected topic for tag number handler: %s", topic)
			}
		},
	)(msg)
}
