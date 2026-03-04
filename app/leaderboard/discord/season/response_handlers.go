package season

import (
	"context"
	"fmt"
	"strings"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
		memberLabel := sm.formatSeasonStandingMemberLabel(ctx, payload.GuildID, s.MemberID)
		sb.WriteString(fmt.Sprintf("\n%d. %s - **%d** pts (%d rounds)", i+1, memberLabel, s.TotalPoints, s.RoundsPlayed))
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

// HandleSeasonEnded handles a successful season end response.
func (sm *seasonManager) HandleSeasonEnded(ctx context.Context, payload *leaderboardevents.EndSeasonSuccessPayloadV1) {
	sm.logger.InfoContext(ctx, "Season ended successfully",
		attr.String("guild_id", string(payload.GuildID)))

	// Prefer the configured leaderboard channel for public announcements
	var channelID string

	// 1. Try Cache
	if cfg, err := sm.guildConfigCache.Get(ctx, string(payload.GuildID)); err == nil && cfg.LeaderboardChannelID != "" {
		channelID = cfg.LeaderboardChannelID
	}

	// 2. Try Resolver
	if channelID == "" {
		if cfg, err := sm.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID)); err == nil && cfg.LeaderboardChannelID != "" {
			channelID = cfg.LeaderboardChannelID
		}
	}

	// 3. Fallback to standard resolution (Context -> Cache -> Resolver)
	if channelID == "" {
		channelID = sm.getChannelID(ctx, string(payload.GuildID))
	}

	if channelID == "" {
		sm.logger.WarnContext(ctx, "No channel ID found to send season end message")
		return
	}

	msg := "🏁 **Season Ended!**\nThe current season has been deactivated. Use `/season start` to begin a new one."
	_, err := sm.session.ChannelMessageSend(channelID, msg)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to send season end message", attr.Error(err))
	}
}

// HandleSeasonEndFailed handles a failed season end response.
func (sm *seasonManager) HandleSeasonEndFailed(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
	sm.logger.WarnContext(ctx, "Season end failed",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	channelID := sm.getChannelID(ctx, string(payload.GuildID))
	if channelID == "" {
		return
	}

	msg := fmt.Sprintf("❌ **Failed to end season**\nReason: %s", payload.Reason)
	_, err := sm.session.ChannelMessageSend(channelID, msg)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to send season end failure message", attr.Error(err))
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

func (sm *seasonManager) formatSeasonStandingMemberLabel(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	memberID sharedtypes.DiscordID,
) string {
	rawMemberID := strings.TrimSpace(string(memberID))
	displayName := sm.lookupSeasonMemberDisplayName(ctx, string(guildID), memberID)
	normalizedID := normalizeSeasonDiscordUserID(rawMemberID)

	switch {
	case normalizedID != "":
		// Prefer mentions for real Discord IDs so Discord resolves to @name.
		return fmt.Sprintf("<@%s>", normalizedID)
	case rawMemberID != "":
		return formatRawSeasonMemberLabel(rawMemberID)
	case displayName != "":
		return formatRawSeasonMemberLabel(displayName)
	default:
		return formatRawSeasonMemberLabel(rawMemberID)
	}
}

func (sm *seasonManager) lookupSeasonMemberDisplayName(
	ctx context.Context,
	guildID string,
	memberID sharedtypes.DiscordID,
) string {
	normalizedID := normalizeSeasonDiscordUserID(string(memberID))
	if guildID == "" || normalizedID == "" {
		return ""
	}

	member, err := sm.session.GuildMember(guildID, normalizedID)
	if err != nil || member == nil {
		return ""
	}

	if displayName := strings.TrimSpace(member.Nick); displayName != "" {
		return sanitizeSeasonDisplayName(displayName)
	}
	if member.User == nil {
		return ""
	}
	if displayName := strings.TrimSpace(member.User.GlobalName); displayName != "" {
		return sanitizeSeasonDisplayName(displayName)
	}
	if displayName := strings.TrimSpace(member.User.Username); displayName != "" {
		return sanitizeSeasonDisplayName(displayName)
	}
	return ""
}

func formatSeasonMemberMention(memberID sharedtypes.DiscordID) string {
	normalizedID := normalizeSeasonDiscordUserID(string(memberID))
	if normalizedID == "" {
		return ""
	}
	return fmt.Sprintf("<@%s>", normalizedID)
}

func normalizeSeasonDiscordUserID(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	if strings.HasPrefix(candidate, "<@") && strings.HasSuffix(candidate, ">") {
		candidate = strings.TrimSuffix(strings.TrimPrefix(candidate, "<@"), ">")
	}

	candidate = strings.TrimPrefix(candidate, "!")
	candidate = strings.TrimPrefix(candidate, "@")
	if candidate == "" || strings.ContainsAny(candidate, "<>@ \t\r\n") {
		return ""
	}
	if !sharedtypes.DiscordID(candidate).IsValid() || !isLikelySeasonDiscordSnowflake(candidate) {
		return ""
	}
	return candidate
}

func isLikelySeasonDiscordSnowflake(candidate string) bool {
	const (
		minSnowflakeLen = 17
		maxSnowflakeLen = 20
	)

	length := len(candidate)
	return length >= minSnowflakeLen && length <= maxSnowflakeLen
}

func formatRawSeasonMemberLabel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "@unknown-user"
	}
	if strings.HasPrefix(trimmed, "<@") && strings.HasSuffix(trimmed, ">") {
		return sanitizeSeasonDisplayName(trimmed)
	}

	sanitized := sanitizeSeasonDisplayName(trimmed)
	if strings.HasPrefix(sanitized, "@") {
		return sanitized
	}
	return fmt.Sprintf("@%s", sanitized)
}

func sanitizeSeasonDisplayName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	normalizedWhitespace := strings.Join(strings.Fields(trimmed), " ")
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`_`, `\_`,
		"`", "\\`",
		`~`, `\~`,
		`|`, `\|`,
	)
	return replacer.Replace(normalizedWhitespace)
}
