package leaderboardupdated

import (
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (lum *leaderboardUpdateManager) resolveLeaderboardDisplayNames(
	ctx context.Context,
	channelID string,
	leaderboard []LeaderboardEntry,
) []LeaderboardEntry {
	if !leaderboardHasMissingDisplayNames(leaderboard) {
		return leaderboard
	}

	guildID := lum.lookupGuildIDForChannel(channelID)
	if guildID == "" {
		return leaderboard
	}

	resolved := make([]LeaderboardEntry, len(leaderboard))
	copy(resolved, leaderboard)

	displayNameByUserID := make(map[string]string, len(resolved))
	for i, entry := range resolved {
		if sanitizeDisplayName(entry.DisplayName) != "" {
			continue
		}

		normalizedUserID := normalizeDiscordUserID(string(entry.UserID))
		if normalizedUserID == "" {
			continue
		}

		if displayName, ok := displayNameByUserID[normalizedUserID]; ok {
			resolved[i].DisplayName = displayName
			continue
		}

		displayName := lum.lookupLeaderboardMemberDisplayName(ctx, guildID, normalizedUserID)
		displayNameByUserID[normalizedUserID] = displayName
		resolved[i].DisplayName = displayName
	}

	return resolved
}

func leaderboardHasMissingDisplayNames(leaderboard []LeaderboardEntry) bool {
	for _, entry := range leaderboard {
		if sanitizeDisplayName(entry.DisplayName) == "" && normalizeDiscordUserID(string(entry.UserID)) != "" {
			return true
		}
	}
	return false
}

func (lum *leaderboardUpdateManager) lookupGuildIDForChannel(channelID string) string {
	if lum == nil || lum.session == nil || strings.TrimSpace(channelID) == "" {
		return ""
	}

	channel, err := lum.session.GetChannel(channelID)
	if err != nil || channel == nil {
		return ""
	}

	return strings.TrimSpace(channel.GuildID)
}

func (lum *leaderboardUpdateManager) lookupLeaderboardMemberDisplayName(
	ctx context.Context,
	guildID string,
	userID string,
) string {
	if lum == nil || lum.session == nil || guildID == "" || userID == "" {
		return ""
	}

	member, err := lum.session.GuildMember(guildID, userID)
	if err != nil || member == nil {
		return ""
	}

	return sanitizeDisplayName(preferredLeaderboardMemberDisplayName(member))
}

func preferredLeaderboardMemberDisplayName(member *discordgo.Member) string {
	if member == nil || member.User == nil {
		return ""
	}
	if displayName := strings.TrimSpace(member.Nick); displayName != "" {
		return displayName
	}
	if displayName := strings.TrimSpace(member.User.GlobalName); displayName != "" {
		return displayName
	}
	return strings.TrimSpace(member.User.Username)
}
