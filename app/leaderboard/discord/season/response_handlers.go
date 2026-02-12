package season

import (
	"context"
	"fmt"
	"strings"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

type contextKey string

const channelIDKey contextKey = "channel_id"

// HandleSeasonStarted handles a successful season start response.
func (sm *seasonManager) HandleSeasonStarted(ctx context.Context, payload *leaderboardevents.StartNewSeasonSuccessPayloadV1) {
	sm.logger.InfoContext(ctx, "Season started successfully",
		attr.String("season_id", payload.SeasonID),
		attr.String("season_name", payload.SeasonName),
		attr.String("guild_id", string(payload.GuildID)))

	channelID := sm.getChannelID(ctx, string(payload.GuildID))
	if channelID == "" {
		sm.logger.WarnContext(ctx, "No channel ID found to send season start message")
		return
	}

	msg := fmt.Sprintf("🎉 **New Season Started!**\n**%s** is now active.", payload.SeasonName)
	_, err := sm.session.ChannelMessageSend(channelID, msg)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to send season start message", attr.Error(err))
	}
}

// HandleSeasonStartFailed handles a failed season start response.
func (sm *seasonManager) HandleSeasonStartFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	sm.logger.WarnContext(ctx, "Season start failed",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	channelID := sm.getChannelID(ctx, string(payload.GuildID))
	if channelID == "" {
		return
	}

	msg := fmt.Sprintf("❌ **Failed to start season**\nReason: %s", payload.Reason)
	_, err := sm.session.ChannelMessageSend(channelID, msg)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to send season start failure message", attr.Error(err))
	}
}

// HandleSeasonStandings handles a successful season standings response.
func (sm *seasonManager) HandleSeasonStandings(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1) {
	sm.logger.InfoContext(ctx, "Received season standings",
		attr.String("season_id", payload.SeasonID),
		attr.String("guild_id", string(payload.GuildID)),
		attr.Int("standings_count", len(payload.Standings)))

	channelID := sm.getChannelID(ctx, string(payload.GuildID))

	if len(payload.Standings) == 0 {
		sm.logger.InfoContext(ctx, "No standings data available for season",
			attr.String("season_id", payload.SeasonID))
		if channelID != "" {
			_, err := sm.session.ChannelMessageSend(channelID, fmt.Sprintf("No standings data available for season ID: %s", payload.SeasonID))
			if err != nil {
				sm.logger.ErrorContext(ctx, "Failed to send no standings message", attr.Error(err))
			}
		}
		return
	}

	// Build a standings summary for logging and discord
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Season Standings** (ID: %s)\n", payload.SeasonID))
	for i, s := range payload.Standings {
		if i >= 10 { // Limit to top 10 for Discord message to avoid length limits
			sb.WriteString(fmt.Sprintf("\n... and %d more", len(payload.Standings)-10))
			break
		}
		sb.WriteString(fmt.Sprintf("\n%d. <@%s> - **%d** pts (%d rounds)", i+1, string(s.MemberID), s.TotalPoints, s.RoundsPlayed))
	}
	sm.logger.InfoContext(ctx, sb.String())

	if channelID != "" {
		_, err := sm.session.ChannelMessageSend(channelID, sb.String())
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to send standings message", attr.Error(err))
		}
	} else {
		sm.logger.WarnContext(ctx, "No channel ID found to send standings")
	}
}

// HandleSeasonStandingsFailed handles a failed season standings retrieval.
func (sm *seasonManager) HandleSeasonStandingsFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	sm.logger.WarnContext(ctx, "Season standings retrieval failed",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	channelID := sm.getChannelID(ctx, string(payload.GuildID))
	if channelID == "" {
		return
	}

	msg := fmt.Sprintf("❌ **Failed to retrieve standings**\nReason: %s", payload.Reason)
	_, err := sm.session.ChannelMessageSend(channelID, msg)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to send standings failure message", attr.Error(err))
	}
}

// getChannelID attempts to retrieve the channel ID from context or falls back to guild config.
func (sm *seasonManager) getChannelID(ctx context.Context, guildID string) string {
	// 1. Try to get from context (if middleware propagated it)
	if val := ctx.Value(channelIDKey); val != nil {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}

	// 2. Fallback to Guild Config (Leaderboard Channel)
	cfg, err := sm.guildConfigCache.Get(ctx, guildID)
	if err == nil && cfg.LeaderboardChannelID != "" {
		return cfg.LeaderboardChannelID
	}

	// 3. Fallback to resolver if cache missed
	config, err := sm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
	if err == nil && config.LeaderboardChannelID != "" {
		return config.LeaderboardChannelID
	}

	return ""
}
