package leaderboardhandlers

import (
	"context"
	"fmt"
	"sort"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
)

// HandleLeaderboardUpdated handles the LeaderboardUpdated event by sending an embedded leaderboard message.
func (h *LeaderboardHandlers) HandleLeaderboardUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleLeaderboardUpdated",
		&leaderboardevents.LeaderboardUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatedPayload := payload.(*leaderboardevents.LeaderboardUpdatedPayload)

			h.Logger.InfoContext(ctx, "Handling leaderboard updated event", attr.CorrelationIDFromMsg(msg))

			if len(updatedPayload.LeaderboardData) == 0 {
				h.Logger.WarnContext(ctx, "Received empty leaderboard data", attr.CorrelationIDFromMsg(msg))
				return nil, nil
			}

			channelID := h.Config.Discord.ChannelID
			if channelID == "" {
				err := fmt.Errorf("missing Discord Channel ID in config")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg))
				return nil, err
			}

			// Convert leaderboard data
			var leaderboardEntries []leaderboardupdated.LeaderboardEntry
			for rank, userID := range updatedPayload.LeaderboardData {
				leaderboardEntries = append(leaderboardEntries, leaderboardupdated.LeaderboardEntry{
					Rank:   sharedtypes.TagNumber(rank),
					UserID: sharedtypes.DiscordID(userID),
				})
			}

			// Sort by rank
			sort.Slice(leaderboardEntries, func(i, j int) bool {
				return leaderboardEntries[i].Rank < leaderboardEntries[j].Rank
			})

			// Send the leaderboard embed
			result, err := h.LeaderboardDiscord.GetLeaderboardUpdateManager().SendLeaderboardEmbed(
				ctx,
				channelID,
				leaderboardEntries,
				int32(1), // Convert to int32
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send leaderboard embed", attr.Error(err))
				return nil, err
			}
			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Error in result from SendLeaderboardEmbed", attr.Error(result.Error))
				return nil, result.Error
			}

			h.Logger.InfoContext(ctx, "Successfully sent leaderboard embed", attr.CorrelationIDFromMsg(msg))

			// Create a trace event
			tracePayload := map[string]interface{}{
				"event_type":  "leaderboard_updated",
				"status":      "embed_sent",
				"channel_id":  channelID,
				"entry_count": len(updatedPayload.LeaderboardData),
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, leaderboardevents.LeaderboardTraceEvent)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event", attr.Error(err))
				return []*message.Message{}, nil
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
