package leaderboardupdated

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

const (
	leaderboardEmbedTitle   = "🏆 Leaderboard"
	maxDescriptionLength    = 4096
	maxDescriptionTruncMark = "\n*(list truncated — too many entries to display)*"
)

type LeaderboardEntry struct {
	Rank         sharedtypes.TagNumber `json:"rank"`
	UserID       sharedtypes.DiscordID `json:"user_id"`
	DisplayName  string                `json:"display_name,omitempty"`
	TotalPoints  int                   `json:"total_points"`
	RoundsPlayed int                   `json:"rounds_played"`
}

// buildLeaderboardDescription formats all leaderboard entries as a single
// description string, capped at Discord's 4096-char description limit.
// No pagination — entries beyond the character cap are silently truncated
// with a note.
func buildLeaderboardDescription(leaderboard []LeaderboardEntry) string {
	if len(leaderboard) == 0 {
		return "*No entries yet.*"
	}

	totalEntries := len(leaderboard)
	var sb strings.Builder

	for i, entry := range leaderboard {
		position := i + 1
		userLabel := formatLeaderboardUser(entry)

		var emoji string
		switch {
		case position == 1:
			emoji = "🥇"
		case position == 2:
			emoji = "🥈"
		case position == 3:
			emoji = "🥉"
		case position == totalEntries && totalEntries > 1:
			emoji = "🗑️"
		default:
			emoji = "🏷️"
		}

		var line string
		if entry.TotalPoints > 0 {
			line = fmt.Sprintf("%s **Tag #%-3d** %s • %d pts (%d rds)\n", emoji, entry.Rank, userLabel, entry.TotalPoints, entry.RoundsPlayed)
		} else {
			line = fmt.Sprintf("%s **Tag #%-3d** %s\n", emoji, entry.Rank, userLabel)
		}

		// Check if adding this line would exceed the limit (leave room for truncation mark)
		if sb.Len()+len(line) > maxDescriptionLength-len(maxDescriptionTruncMark) {
			sb.WriteString(maxDescriptionTruncMark)
			return sb.String()
		}

		sb.WriteString(line)
	}

	return sb.String()
}

func formatLeaderboardUser(entry LeaderboardEntry) string {
	rawUserID := strings.TrimSpace(string(entry.UserID))
	displayName := sanitizeDisplayName(entry.DisplayName)
	normalizedID := normalizeDiscordUserID(rawUserID)

	switch {
	case normalizedID != "":
		// Real Discord IDs should always render as mentions so Discord resolves
		// to the current server-visible @name.
		return fmt.Sprintf("<@%s>", normalizedID)
	case rawUserID != "":
		// For human-readable handles (non-Discord IDs), keep the original @label.
		// But prefer enriched display names for legacy placeholder/numeric IDs.
		if displayName != "" && (isNumericLeaderboardID(rawUserID) || isPlaceholderLeaderboardLabel(rawUserID)) {
			return formatRawLeaderboardUserLabel(displayName)
		}
		return formatRawLeaderboardUserLabel(rawUserID)
	case displayName != "":
		return formatRawLeaderboardUserLabel(displayName)
	default:
		return formatRawLeaderboardUserLabel(rawUserID)
	}
}

func formatUserMention(userID sharedtypes.DiscordID) string {
	normalizedID := normalizeDiscordUserID(string(userID))
	if normalizedID == "" {
		return ""
	}
	return fmt.Sprintf("<@%s>", normalizedID)
}

func normalizeDiscordUserID(raw string) string {
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
	if !sharedtypes.DiscordID(candidate).IsValid() || !isLikelyDiscordSnowflake(candidate) {
		return ""
	}

	return candidate
}

func isLikelyDiscordSnowflake(candidate string) bool {
	const (
		minSnowflakeLen = 17
		maxSnowflakeLen = 20
	)

	length := len(candidate)
	return length >= minSnowflakeLen && length <= maxSnowflakeLen
}

func isNumericLeaderboardID(raw string) bool {
	if raw == "" {
		return false
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isPlaceholderLeaderboardLabel(raw string) bool {
	return strings.Contains(strings.ToLower(raw), "placeholder")
}

func formatRawLeaderboardUserLabel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "@unknown-user"
	}

	if strings.HasPrefix(trimmed, "<@") && strings.HasSuffix(trimmed, ">") {
		return sanitizeDisplayName(trimmed)
	}

	sanitized := sanitizeDisplayName(trimmed)
	if strings.HasPrefix(sanitized, "@") {
		return sanitized
	}

	return fmt.Sprintf("@%s", sanitized)
}

func sanitizeDisplayName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	normalizedWhitespace := strings.Join(strings.Fields(trimmed), " ")
	return escapeDiscordMarkdown(normalizedWhitespace)
}

func escapeDiscordMarkdown(raw string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`_`, `\_`,
		"`", "\\`",
		`~`, `\~`,
		`|`, `\|`,
	)
	return replacer.Replace(raw)
}

// buildLeaderboardEmbed constructs the embed with all entries in the description.
// The page parameter is accepted for interface compatibility but ignored.
func buildLeaderboardEmbed(leaderboard []LeaderboardEntry, _ int32) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	desc := buildLeaderboardDescription(leaderboard)

	embed := &discordgo.MessageEmbed{
		Title:       leaderboardEmbedTitle,
		Description: desc,
		Color:       0xFFD700, // Gold
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Frolf Leaderboard • Updated: %s", time.Now().Format(time.RFC1123)),
		},
	}

	// No pagination buttons — the embed is a shared channel message and
	// editing it would change the view for everyone.
	return embed, nil
}

func (lum *leaderboardUpdateManager) SendLeaderboardEmbed(ctx context.Context, channelID string, leaderboard []LeaderboardEntry, page int32) (LeaderboardUpdateOperationResult, error) {
	return lum.operationWrapper(ctx, "send_leaderboard_embed", func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
		embed, _ := buildLeaderboardEmbed(leaderboard, page)

		if existingMessageID := lum.getTrackedMessageID(channelID); existingMessageID != "" {
			editedMessage, err := lum.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:      existingMessageID,
				Channel: channelID,
				Embeds:  &[]*discordgo.MessageEmbed{embed},
				// Clear any old pagination components
				Components: &[]discordgo.MessageComponent{},
			})
			if err == nil {
				return LeaderboardUpdateOperationResult{Success: editedMessage}, nil
			}
			if isUnknownMessageError(err) {
				lum.clearTrackedMessageID(channelID)
			} else {
				err := fmt.Errorf("failed to update persistent leaderboard message: %w", err)
				lum.logger.ErrorContext(ctx, err.Error())
				return LeaderboardUpdateOperationResult{Error: err}, err
			}
		}

		if discoveredMessageID, err := lum.findExistingLeaderboardMessage(ctx, channelID); err != nil {
			lum.logger.WarnContext(ctx, "Failed to discover existing leaderboard message, will post a new one", "channel_id", channelID, "error", err)
		} else if discoveredMessageID != "" {
			lum.setTrackedMessageID(channelID, discoveredMessageID)
			editedMessage, editErr := lum.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         discoveredMessageID,
				Channel:    channelID,
				Embeds:     &[]*discordgo.MessageEmbed{embed},
				Components: &[]discordgo.MessageComponent{},
			})
			if editErr == nil {
				return LeaderboardUpdateOperationResult{Success: editedMessage}, nil
			}
			if isUnknownMessageError(editErr) {
				lum.clearTrackedMessageID(channelID)
			} else {
				err := fmt.Errorf("failed to update discovered leaderboard message: %w", editErr)
				lum.logger.ErrorContext(ctx, err.Error())
				return LeaderboardUpdateOperationResult{Error: err}, err
			}
		}

		message, err := lum.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			err := fmt.Errorf("failed to send leaderboard message: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, err
		}

		lum.setTrackedMessageID(channelID, message.ID)
		return LeaderboardUpdateOperationResult{Success: message}, nil
	})
}

func (lum *leaderboardUpdateManager) getTrackedMessageID(channelID string) string {
	lum.messageMu.RLock()
	defer lum.messageMu.RUnlock()
	if lum.messageByChannelID == nil {
		return ""
	}
	return lum.messageByChannelID[channelID]
}

func (lum *leaderboardUpdateManager) setTrackedMessageID(channelID, messageID string) {
	if channelID == "" || messageID == "" {
		return
	}
	lum.messageMu.Lock()
	defer lum.messageMu.Unlock()
	if lum.messageByChannelID == nil {
		lum.messageByChannelID = make(map[string]string)
	}
	lum.messageByChannelID[channelID] = messageID
}

func (lum *leaderboardUpdateManager) clearTrackedMessageID(channelID string) {
	lum.messageMu.Lock()
	defer lum.messageMu.Unlock()
	if lum.messageByChannelID == nil {
		return
	}
	delete(lum.messageByChannelID, channelID)
}

func (lum *leaderboardUpdateManager) findExistingLeaderboardMessage(ctx context.Context, channelID string) (string, error) {
	if channelID == "" {
		return "", nil
	}

	messages, err := lum.session.ChannelMessages(channelID, 50, "", "", "")
	if err != nil {
		return "", err
	}

	botID := ""
	if botUser, botErr := lum.session.GetBotUser(); botErr == nil && botUser != nil {
		botID = botUser.ID
	}

	for _, message := range messages {
		if message == nil || len(message.Embeds) == 0 {
			continue
		}
		if botID != "" && message.Author != nil && message.Author.ID != botID {
			continue
		}
		embed := message.Embeds[0]
		if embed != nil && embed.Title == leaderboardEmbedTitle {
			lum.logger.InfoContext(ctx, "Discovered existing leaderboard message to reuse", "channel_id", channelID, "message_id", message.ID)
			return message.ID, nil
		}
	}

	return "", nil
}

func isUnknownMessageError(err error) bool {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) && restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unknown message")
}
