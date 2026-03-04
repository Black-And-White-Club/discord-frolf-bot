package handlers

import (
	"context"
	"sort"
	"strconv"
	"strings"

	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleLeaderboardRetrieveRequest handles a leaderboard retrieve request event from Discord.
func (h *LeaderboardHandlers) HandleLeaderboardRetrieveRequest(ctx context.Context,
	payload *discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard retrieve request")

	discordPayload := payload

	// Convert to backend payload
	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(discordPayload.GuildID),
	}

	h.logger.InfoContext(ctx, "Successfully processed leaderboard retrieve request",
		attr.String("guild_id", discordPayload.GuildID))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.GetLeaderboardRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleLeaderboardUpdatedNotification handles backend.leaderboard.updated and re-requests full leaderboard.
func (h *LeaderboardHandlers) HandleLeaderboardUpdatedNotification(ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard updated notification")

	updatePayload := payload

	backendPayload := leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: updatePayload.GuildID,
	}

	h.logger.InfoContext(ctx, "Requesting full leaderboard after update notification",
		attr.String("guild_id", string(updatePayload.GuildID)))

	return []handlerwrapper.Result{
		{
			Topic:   leaderboardevents.GetLeaderboardRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleLeaderboardResponse handles backend.leaderboard.get.response and translates to Discord response.
func (h *LeaderboardHandlers) HandleLeaderboardResponse(ctx context.Context,
	payload *leaderboardevents.GetLeaderboardResponsePayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling leaderboard response")

	payloadData := payload

	leaderboardData := make([]leaderboardtypes.LeaderboardEntry, len(payloadData.Leaderboard))
	for i, entry := range payloadData.Leaderboard {
		leaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			TagNumber:    entry.TagNumber,
			UserID:       entry.UserID,
			TotalPoints:  entry.TotalPoints,
			RoundsPlayed: entry.RoundsPlayed,
		}
	}

	discordPayload := discordleaderboardevents.LeaderboardRetrievedPayloadV1{
		Leaderboard: leaderboardData,
		GuildID:     string(payloadData.GuildID),
	}

	channelID := h.resolveLeaderboardChannelID(ctx, string(payloadData.GuildID))
	if channelID != "" && h.service != nil {
		entries := make([]leaderboardupdated.LeaderboardEntry, 0, len(leaderboardData))
		for _, entry := range leaderboardData {
			entries = append(entries, leaderboardupdated.LeaderboardEntry{
				Rank:         entry.TagNumber,
				UserID:       entry.UserID,
				DisplayName:  resolveProfileDisplayName(entry.TagNumber, entry.UserID, payloadData.Profiles),
				TotalPoints:  entry.TotalPoints,
				RoundsPlayed: entry.RoundsPlayed,
			})
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Rank < entries[j].Rank
		})

		manager := h.service.GetLeaderboardUpdateManager()
		if manager != nil {
			result, err := manager.SendLeaderboardEmbed(ctx, channelID, entries, 1)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to send leaderboard embed from full snapshot",
					attr.Error(err),
					attr.String("guild_id", string(payloadData.GuildID)),
					attr.String("channel_id", channelID),
				)
				return nil, err
			}
			if result.Error != nil {
				h.logger.ErrorContext(ctx, "Failed to send leaderboard embed from full snapshot",
					attr.Error(result.Error),
					attr.String("guild_id", string(payloadData.GuildID)),
					attr.String("channel_id", channelID),
				)
				return nil, result.Error
			}
		}
	}

	h.logger.InfoContext(ctx, "Successfully processed leaderboard data",
		attr.String("guild_id", string(payloadData.GuildID)),
		attr.Int("entry_count", len(leaderboardData)))

	return []handlerwrapper.Result{
		{
			Topic:   discordleaderboardevents.LeaderboardRetrievedV1,
			Payload: discordPayload,
		},
	}, nil
}

func (h *LeaderboardHandlers) resolveLeaderboardChannelID(ctx context.Context, guildID string) string {
	// 1. Guild config (authoritative)
	if h.guildConfigResolver != nil && guildID != "" {
		if guildCfg, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && guildCfg != nil && guildCfg.LeaderboardChannelID != "" {
			return guildCfg.LeaderboardChannelID
		} else if err != nil {
			h.logger.WarnContext(ctx, "Failed to get guild config, will try static config",
				attr.Error(err),
				attr.String("guild_id", guildID),
			)
		}
	}

	// 2. Static config fallback (single-tenant / tests)
	if h.config != nil && h.config.Discord.LeaderboardChannelID != "" {
		return h.config.Discord.LeaderboardChannelID
	}

	h.logger.WarnContext(ctx, "No leaderboard channel could be resolved (guild config + static config empty)",
		attr.String("guild_id", guildID))
	return ""
}

func resolveProfileDisplayName(
	tagNumber sharedtypes.TagNumber,
	userID sharedtypes.DiscordID,
	profiles map[sharedtypes.DiscordID]*usertypes.UserProfile,
) string {
	if len(profiles) == 0 {
		return ""
	}

	if profile, ok := profiles[userID]; ok && profile != nil {
		if displayName := preferredProfileDisplayName(profile); displayName != "" {
			return displayName
		}
	}

	normalizedUserID := normalizeDiscordUserIDForProfileLookup(string(userID))
	if placeholderTagID := extractPlaceholderTagID(string(userID)); placeholderTagID != "" {
		if profile, ok := profiles[sharedtypes.DiscordID(placeholderTagID)]; ok && profile != nil {
			if displayName := preferredProfileDisplayName(profile); displayName != "" {
				return displayName
			}
		}
	}

	if tagNumber > 0 {
		tagProfileID := strconv.Itoa(int(tagNumber))
		if profile, ok := profiles[sharedtypes.DiscordID(tagProfileID)]; ok && profile != nil {
			if displayName := preferredProfileDisplayName(profile); displayName != "" {
				return displayName
			}
		}
	}

	if normalizedUserID == "" {
		return ""
	}

	if profile, ok := profiles[sharedtypes.DiscordID(normalizedUserID)]; ok && profile != nil {
		if displayName := preferredProfileDisplayName(profile); displayName != "" {
			return displayName
		}
	}

	for profileUserID, profile := range profiles {
		if profile == nil {
			continue
		}
		if normalizeDiscordUserIDForProfileLookup(string(profileUserID)) != normalizedUserID {
			continue
		}
		if displayName := preferredProfileDisplayName(profile); displayName != "" {
			return displayName
		}
	}

	return ""
}

func extractPlaceholderTagID(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" || !strings.Contains(lower, "placeholder") {
		return ""
	}

	for _, field := range strings.Fields(lower) {
		candidate := strings.TrimPrefix(strings.TrimSpace(field), "#")
		if candidate == "" {
			continue
		}
		if _, err := strconv.Atoi(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func normalizeDiscordUserIDForProfileLookup(raw string) string {
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

	return candidate
}

func preferredProfileDisplayName(profile *usertypes.UserProfile) string {
	if profile == nil {
		return ""
	}

	displayName := strings.TrimSpace(profile.DisplayName)
	if profile.UDiscUsername != nil {
		if username := strings.TrimSpace(*profile.UDiscUsername); username != "" {
			if displayName == "" || strings.Contains(strings.ToLower(displayName), "placeholder") {
				return username
			}
		}
	}
	if displayName != "" {
		return displayName
	}
	if profile.UDiscName != nil {
		if name := strings.TrimSpace(*profile.UDiscName); name != "" {
			return name
		}
	}

	return ""
}
