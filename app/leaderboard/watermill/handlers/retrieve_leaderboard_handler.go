package leaderboardhandlers

// import (
// 	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// func (h *LeaderboardHandlers) HandleLeaderboardRetrieveRequest(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling leaderboard retrieve request", attr.CorrelationIDFromMsg(msg))
// 	var payload discordleaderboardevents.LeaderboardRetrieveRequestPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
// 		return nil, err
// 	}
// 	backendPayload := leaderboardevents.GetLeaderboardRequestPayload{}
// 	backendMsg, err := h.Helper.CreateResultMessage(msg, backendPayload, leaderboardevents.GetLeaderboardRequest)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, err
// 	}
// 	h.Logger.Info(ctx, "Successfully processed leaderboard retrieve request", attr.CorrelationIDFromMsg(msg))
// 	return []*message.Message{backendMsg}, nil
// }

// // HandleLeaderboardData handles both backend.leaderboard.get.response AND backend.leaderboard.updated.
// func (h *LeaderboardHandlers) HandleLeaderboardData(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling leaderboard data", attr.CorrelationIDFromMsg(msg), attr.Topic(msg.Metadata.Get("topic")))
// 	topic := msg.Metadata.Get("topic")
// 	if topic == leaderboardevents.LeaderboardUpdated {
// 		backendPayload := leaderboardevents.GetLeaderboardRequestPayload{}
// 		backendMsg, err := h.Helper.CreateResultMessage(msg, backendPayload, leaderboardevents.GetLeaderboardRequest)
// 		if err != nil {
// 			h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 			return nil, err
// 		}
// 		h.Logger.Info(ctx, "Requesting full leaderboard after update notification", attr.CorrelationIDFromMsg(msg))
// 		return []*message.Message{backendMsg}, nil
// 	}
// 	var payload leaderboardevents.GetLeaderboardResponsePayload
// 	if err := h.Helper.UnmarshalPayload(msg, &payload); err != nil {
// 		return nil, err
// 	}
// 	leaderboardData := make([]leaderboardevents.LeaderboardEntry, len(payload.Leaderboard))
// 	for i, entry := range payload.Leaderboard {
// 		leaderboardData[i] = leaderboardevents.LeaderboardEntry{
// 			TagNumber: entry.TagNumber,
// 			DiscordID: entry.DiscordID,
// 		}
// 	}
// 	// Create the Discord payload.
// 	discordPayload := discordleaderboardevents.LeaderboardRetrievedPayload{
// 		Leaderboard: leaderboardData,
// 	}
// 	// Create the outgoing message.
// 	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardRetrievedTopic)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, err
// 	}
// 	h.Logger.Info(ctx, "Successfully processed leaderboard data", attr.CorrelationIDFromMsg(msg), attr.Topic(msg.Metadata.Get("topic")))
// 	return []*message.Message{discordMsg}, nil
// }
