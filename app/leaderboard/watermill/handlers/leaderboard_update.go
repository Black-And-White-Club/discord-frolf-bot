package leaderboardhandlers

import (
	"context"
	"sort"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
)

// HandleBatchTagAssigned handles batch tag assignment completions and sends leaderboard embed
func (h *LeaderboardHandlers) HandleBatchTagAssigned(ctx context.Context,
	payload interface{}) ([]handlerwrapper.Result, error) {
	h.Logger.InfoContext(ctx, "Handling batch tag assigned event")

	batchPayload := payload.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1)

	guildID := string(batchPayload.GuildID)

	if len(batchPayload.Assignments) == 0 {
		h.Logger.WarnContext(ctx, "Received empty batch assignment data",
			attr.String("guild_id", guildID))
		return []handlerwrapper.Result{}, nil
	}

	// Resolve target channel with safer precedence:
	// 1. Guild config (authoritative)
	// 2. Static config fallback
	var channelID string

	// Try guild-specific config via resolver first
	if h.GuildConfigResolver != nil && guildID != "" {
		if guildCfg, err := h.GuildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && guildCfg != nil && guildCfg.LeaderboardChannelID != "" {
			channelID = guildCfg.LeaderboardChannelID
		} else if err != nil {
			h.Logger.WarnContext(ctx, "Failed to get guild config, will try static config",
				attr.Error(err),
				attr.String("guild_id", guildID),
			)
		}
	}

	// Static config fallback (single-tenant / tests)
	if channelID == "" && h.Config != nil && h.Config.Discord.LeaderboardChannelID != "" {
		channelID = h.Config.Discord.LeaderboardChannelID
	}

	// If still missing, emit trace and return early
	if channelID == "" {
		h.Logger.ErrorContext(ctx,
			"No leaderboard channel could be resolved (guild config + static config empty)",
			attr.String("guild_id", guildID),
			attr.Int("assignment_count", batchPayload.AssignmentCount),
		)

		tracePayload := map[string]interface{}{
			"event_type":       "batch_assignment_completed",
			"status":           "missing_channel_id",
			"entry_count":      len(batchPayload.Assignments),
			"batch_id":         batchPayload.BatchID,
			"guild_id":         guildID,
			"assignment_count": batchPayload.AssignmentCount,
		}

		return []handlerwrapper.Result{
			{
				Topic:   leaderboardevents.LeaderboardTraceEvent,
				Payload: tracePayload,
				Metadata: map[string]string{
					"guild_id": guildID,
				},
			},
		}, nil
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
		int32(1),
	)
	if err != nil {
		h.Logger.ErrorContext(ctx, "Failed to send leaderboard embed",
			attr.Error(err),
			attr.String("guild_id", guildID),
			attr.String("channel_id", channelID),
		)
		return nil, err
	}
	if result.Error != nil {
		h.Logger.ErrorContext(ctx, "Error in result from SendLeaderboardEmbed",
			attr.Error(result.Error),
			attr.String("guild_id", guildID),
		)
		return nil, result.Error
	}

	h.Logger.InfoContext(ctx,
		"Successfully sent leaderboard embed for batch assignment",
		attr.Int("assignment_count", batchPayload.AssignmentCount),
		attr.String("batch_id", batchPayload.BatchID),
		attr.String("guild_id", guildID),
		attr.String("channel_id", channelID),
	)

	// We don't return a trace event here because returning a Result causes
	// Watermill to attempt to publish it. If no consumer/stream exists for
	// that topic, the publish can fail and the input message is Nacked,
	// causing the handler (and its side-effects) to be retried. Returning
	// an empty result set ensures the original message is Acked after
	// successful handling and prevents duplicate Discord posts.
	return []handlerwrapper.Result{}, nil
}
