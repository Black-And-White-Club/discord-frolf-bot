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
			line = fmt.Sprintf("%s **Tag #%-3d** <@%s> • %d pts (%d rds)\n", emoji, entry.Rank, entry.UserID, entry.TotalPoints, entry.RoundsPlayed)
		} else {
			line = fmt.Sprintf("%s **Tag #%-3d** <@%s>\n", emoji, entry.Rank, entry.UserID)
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
