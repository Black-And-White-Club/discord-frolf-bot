package leaderboardhandlers

import (
	"context"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleLeaderboardRetrieveRequest handles a leaderboard retrieve request event from Discord.
func (h *LeaderboardHandlers) HandleLeaderboardRetrieveRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleLeaderboardRetrieveRequest",
		&discordleaderboardevents.LeaderboardRetrieveRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			h.Logger.InfoContext(ctx, "Handling leaderboard retrieve request", attr.CorrelationIDFromMsg(msg))

			discordPayload := payload.(*discordleaderboardevents.LeaderboardRetrieveRequestPayload)

			// Convert to backend payload
			backendPayload := leaderboardevents.GetLeaderboardRequestPayload{
				GuildID: sharedtypes.GuildID(discordPayload.GuildID),
			}

			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.GetLeaderboardRequest)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend message", attr.Error(err))
				return nil, err
			}

			h.Logger.InfoContext(ctx, "Successfully processed leaderboard retrieve request", attr.CorrelationIDFromMsg(msg))
			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleLeaderboardData handles both backend.leaderboard.get.response AND backend.leaderboard.updated.
func (h *LeaderboardHandlers) HandleLeaderboardData(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleLeaderboardData",
		nil, // No unmarshal up front — we conditionally unmarshal later based on topic
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			topic := msg.Metadata.Get("topic")

			h.Logger.InfoContext(ctx, "Handling leaderboard data", attr.CorrelationIDFromMsg(msg), attr.Topic(topic))

			// Special case: If topic is "LeaderboardUpdated", trigger a re-request
			if topic == leaderboardevents.LeaderboardUpdated {
				// For LeaderboardUpdated, we don't need to unmarshal - just send empty request
				backendPayload := leaderboardevents.GetLeaderboardRequestPayload{}

				backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, leaderboardevents.GetLeaderboardRequest)
				if err != nil {
					h.Logger.ErrorContext(ctx, "Failed to create backend message after update", attr.Error(err))
					return nil, err
				}

				h.Logger.InfoContext(ctx, "Requesting full leaderboard after update notification", attr.CorrelationIDFromMsg(msg))
				return []*message.Message{backendMsg}, nil
			}

			// Otherwise, assume it's a leaderboard response
			var payloadData leaderboardevents.GetLeaderboardResponsePayload
			if err := h.Helpers.UnmarshalPayload(msg, &payloadData); err != nil {
				h.Logger.ErrorContext(ctx, "Failed to unmarshal leaderboard response", attr.Error(err))
				return nil, err
			}

			leaderboardData := make([]leaderboardtypes.LeaderboardEntry, len(payloadData.Leaderboard))
			for i, entry := range payloadData.Leaderboard {
				leaderboardData[i] = leaderboardtypes.LeaderboardEntry{
					TagNumber: entry.TagNumber,
					UserID:    entry.UserID,
				}
			}

			discordPayload := discordleaderboardevents.LeaderboardRetrievedPayload{
				Leaderboard: leaderboardData,
			}

			discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardRetrievedTopic)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create discord message from leaderboard data", attr.Error(err))
				return nil, err
			}

			h.Logger.InfoContext(ctx, "Successfully processed leaderboard data", attr.CorrelationIDFromMsg(msg), attr.Topic(topic))
			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}
