package leaderboardhandlers

import (
	"context"
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

			// Extract guild ID (prefer payload, fallback to metadata)
			guildID := ""
			if batchPayload.GuildID != "" {
				guildID = string(batchPayload.GuildID)
			} else if mdGuild := msg.Metadata.Get("guild_id"); mdGuild != "" {
				guildID = mdGuild
			}

			h.Logger.InfoContext(
				ctx,
				"Handling batch tag assigned event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", guildID),
			)

			if len(batchPayload.Assignments) == 0 {
				h.Logger.WarnContext(ctx, "Received empty batch assignment data", attr.CorrelationIDFromMsg(msg))
				return nil, nil
			}

			// Resolve target channel with safer precedence:
			// 1. Guild config (authoritative)
			// 2. Static config fallback
			// 3. Metadata channel_id ONLY if no configured channel could be resolved AND source is not a discord_claim
			//    (We don't want to spam the slash command channel with full leaderboard embeds.)
			var channelID string
			source := msg.Metadata.Get("source")

			// Try guild-specific config via resolver first
			if h.GuildConfigResolver != nil && guildID != "" {
				if guildCfg, err := h.GuildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && guildCfg != nil && guildCfg.LeaderboardChannelID != "" {
					channelID = guildCfg.LeaderboardChannelID
				} else if err != nil {
					h.Logger.WarnContext(
						ctx,
						"Failed to get guild config, will try static config",
						attr.Error(err),
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", guildID),
					)
				}
			}

			// Static config fallback (single-tenant / tests)
			if channelID == "" && h.Config != nil && h.Config.Discord.LeaderboardChannelID != "" {
				channelID = h.Config.Discord.LeaderboardChannelID
			}

			// As a last resort, if we still don't have a channel and source is NOT a claim, allow metadata override
			if channelID == "" && source != "discord_claim" {
				if mdChan := msg.Metadata.Get("channel_id"); mdChan != "" {
					channelID = mdChan
					h.Logger.DebugContext(ctx, "Using metadata channel_id as fallback", attr.CorrelationIDFromMsg(msg), attr.String("channel_id", channelID))
				}
			}

			// If still missing, emit trace and ACK to avoid retry storms
			if channelID == "" {
				h.Logger.ErrorContext(
					ctx,
					"No leaderboard channel could be resolved (guild config + static config empty and metadata suppressed)",
					attr.CorrelationIDFromMsg(msg),
					attr.String("guild_id", guildID),
					attr.String("source", source),
				)

				tracePayload := map[string]interface{}{
					"event_type":  "batch_assignment_completed",
					"status":      "missing_channel_id",
					"channel_id":  "",
					"entry_count": len(batchPayload.Assignments),
					"batch_id":    batchPayload.BatchID,
					"guild_id":    guildID,
					"source":      source,
				}

				traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, leaderboardevents.LeaderboardTraceEvent)
				if err != nil {
					h.Logger.ErrorContext(ctx, "Failed to create trace event", attr.Error(err))
					return []*message.Message{}, nil
				}
				if guildID != "" {
					traceMsg.Metadata.Set("guild_id", guildID)
				}
				return []*message.Message{traceMsg}, nil
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
			sort.Slice(
				leaderboardEntries, func(i, j int) bool {
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
				h.Logger.ErrorContext(ctx, "Failed to send leaderboard embed", attr.Error(err))
				return nil, err
			}
			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Error in result from SendLeaderboardEmbed", attr.Error(result.Error))
				return nil, result.Error
			}

			h.Logger.InfoContext(
				ctx,
				"Successfully sent leaderboard embed for batch assignment",
				attr.CorrelationIDFromMsg(msg),
				attr.Int("assignment_count", batchPayload.AssignmentCount),
				attr.String("batch_id", batchPayload.BatchID),
				attr.String("guild_id", guildID),
			)

			// Create a trace event
			tracePayload := map[string]interface{}{
				"event_type":  "batch_assignment_completed",
				"status":      "embed_sent",
				"channel_id":  channelID,
				"entry_count": len(batchPayload.Assignments),
				"batch_id":    batchPayload.BatchID,
				"guild_id":    guildID,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, leaderboardevents.LeaderboardTraceEvent)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event", attr.Error(err))
				return []*message.Message{}, nil
			}

			// Propagate guild_id in metadata for downstream consumers
			if guildID != "" {
				traceMsg.Metadata.Set("guild_id", guildID)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
