package leaderboardhandlers

import (
	"fmt"
	"sort"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated" // Import the correct package where `LeaderboardEntry` is defined
)

// HandleLeaderboardUpdated handles the LeaderboardUpdated event by sending an embedded leaderboard message.
func (h *LeaderboardHandlers) HandleLeaderboardUpdated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling leaderboard updated event", attr.CorrelationIDFromMsg(msg))

	// Unmarshal the payload
	var payload leaderboardevents.LeaderboardUpdatedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Validate required fields
	if len(payload.LeaderboardData) == 0 {
		h.Logger.Warn(ctx, "Received empty leaderboard data", attr.CorrelationIDFromMsg(msg))
		return nil, nil // No error, just nothing to process
	}

	// Use ChannelID from config instead of payload
	channelID := h.Config.Discord.ChannelID
	if channelID == "" {
		h.Logger.Error(ctx, "Missing Discord Channel ID in config", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("missing Discord Channel ID in config")
	}

	// Convert leaderboard data to the correct type
	var leaderboardEntries []leaderboardupdated.LeaderboardEntry // Use the correct type
	for rank, userID := range payload.LeaderboardData {
		leaderboardEntries = append(leaderboardEntries, leaderboardupdated.LeaderboardEntry{
			Rank:   rank,
			UserID: userID,
		})
	}

	// Sort leaderboard entries by rank
	sort.Slice(leaderboardEntries, func(i, j int) bool {
		return leaderboardEntries[i].Rank < leaderboardEntries[j].Rank
	})

	// Send the leaderboard embed
	_, err := h.LeaderboardDiscord.GetLeaderboardUpdateManager().SendLeaderboardEmbed(
		channelID, leaderboardEntries, 1, // Always start from page 1
	)
	if err != nil {
		h.Logger.Error(ctx, "Failed to send leaderboard embed", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to send leaderboard embed: %w", err)
	}

	h.Logger.Info(ctx, "Successfully sent leaderboard embed", attr.CorrelationIDFromMsg(msg))

	// Create a trace event
	tracePayload := map[string]interface{}{
		"event_type":  "leaderboard_updated",
		"status":      "embed_sent",
		"channel_id":  channelID,
		"entry_count": len(payload.LeaderboardData),
	}

	traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, leaderboardevents.LeaderboardTraceEvent)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create trace event", attr.Error(err))
		return []*message.Message{}, nil
	}

	return []*message.Message{traceMsg}, nil
}
