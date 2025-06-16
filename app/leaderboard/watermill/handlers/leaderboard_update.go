package leaderboardhandlers

import (
	"context"
	"fmt"
	"sort"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
)

// HandleBatchTagAssigned handles batch tag assignment completions and sends leaderboard embed
func (h *LeaderboardHandlers) HandleBatchTagAssigned(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleBatchTagAssigned",
		&leaderboardevents.BatchTagAssignedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			batchPayload := payload.(*leaderboardevents.BatchTagAssignedPayload)

			h.Logger.InfoContext(ctx, "Handling batch tag assigned event", attr.CorrelationIDFromMsg(msg))

			if len(batchPayload.Assignments) == 0 {
				h.Logger.WarnContext(ctx, "Received empty batch assignment data", attr.CorrelationIDFromMsg(msg))
				return nil, nil
			}

			channelID := h.Config.Discord.LeaderboardChannelID
			if channelID == "" {
				err := fmt.Errorf("missing Discord Channel ID in config")
				h.Logger.ErrorContext(ctx, err.Error(), attr.CorrelationIDFromMsg(msg))
				return nil, err
			}

			// Convert batch assignments to leaderboard entries
			var leaderboardEntries []leaderboardupdated.LeaderboardEntry
			for _, assignment := range batchPayload.Assignments {
				leaderboardEntries = append(leaderboardEntries, leaderboardupdated.LeaderboardEntry{
					Rank:   assignment.TagNumber, // TagNumber is the rank/position
					UserID: assignment.UserID,
				})
			}

			// Sort by rank (tag number)
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

			h.Logger.InfoContext(ctx, "Successfully sent leaderboard embed for batch assignment",
				attr.CorrelationIDFromMsg(msg),
				attr.Int("assignment_count", batchPayload.AssignmentCount),
				attr.String("batch_id", batchPayload.BatchID),
			)

			// Create a trace event
			tracePayload := map[string]interface{}{
				"event_type":  "batch_assignment_completed",
				"status":      "embed_sent",
				"channel_id":  channelID,
				"entry_count": len(batchPayload.Assignments),
				"batch_id":    batchPayload.BatchID,
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
