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

const leaderboardEmbedTitle = "🏆 Leaderboard"

type LeaderboardEntry struct {
	Rank         sharedtypes.TagNumber `json:"rank"`
	UserID       sharedtypes.DiscordID `json:"user_id"`
	TotalPoints  int                   `json:"total_points"`
	RoundsPlayed int                   `json:"rounds_played"`
}

func (lum *leaderboardUpdateManager) SendLeaderboardEmbed(ctx context.Context, channelID string, leaderboard []LeaderboardEntry, page int32) (LeaderboardUpdateOperationResult, error) {
	return lum.operationWrapper(ctx, "send_leaderboard_embed", func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
		const entriesPerPage = int32(10)
		totalPages := (int32(len(leaderboard)) + entriesPerPage - 1) / entriesPerPage

		if totalPages == 0 {
			totalPages = 1
		}

		// Ensure page is within valid range
		if page < 1 {
			page = 1
		} else if page > totalPages {
			page = totalPages
		}

		// Calculate slice range
		start := (page - 1) * entriesPerPage
		end := min(start+entriesPerPage, int32(len(leaderboard)))

		// Build leaderboard fields with ranking emojis
		fields := []*discordgo.MessageEmbedField{}
		totalEntries := len(leaderboard)

		// Only create fields if there are entries to show
		if totalEntries > 0 {
			// Build leaderboard as a single formatted table instead of multiple fields
			var leaderboardText string

			for i, entry := range leaderboard[start:end] {
				// Calculate the actual position in the full leaderboard
				actualPosition := int(start) + i + 1

				// Determine emoji based on position and total entries
				var emoji string
				switch {
				case actualPosition == 1:
					emoji = "🥇" // Gold medal for 1st place
				case actualPosition == 2:
					emoji = "🥈" // Silver medal for 2nd place
				case actualPosition == 3:
					emoji = "🥉" // Bronze medal for 3rd place
				case actualPosition == totalEntries && totalEntries > 1:
					emoji = "🗑️" // Trash can for last place
				default:
					emoji = "🏷️" // Tag emoji for everyone else
				}

				// Format each row with proper spacing
				if entry.TotalPoints > 0 {
					leaderboardText += fmt.Sprintf("%s **Tag #%-3d** <@%s> • %d pts (%d rds)\n", emoji, entry.Rank, entry.UserID, entry.TotalPoints, entry.RoundsPlayed)
				} else {
					leaderboardText += fmt.Sprintf("%s **Tag #%-3d** <@%s>\n", emoji, entry.Rank, entry.UserID)
				}
			}

			// Create a single field with the formatted table
			fields = []*discordgo.MessageEmbedField{
				{
					Name:   "Tags",
					Value:  leaderboardText,
					Inline: false,
				},
			}
		}

		embed := &discordgo.MessageEmbed{
			Title:       leaderboardEmbedTitle,
			Description: fmt.Sprintf("Page %d/%d", page, totalPages),
			Color:       0xFFD700, // Gold color
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Frolf Leaderboard • Updated: %s", time.Now().Format(time.RFC1123)),
			},
		}

		// Create pagination buttons
		components := []discordgo.MessageComponent{}
		if totalPages > 1 {
			components = append(components, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "⬅️ Previous",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("leaderboard_prev|%d", page-1),
						Disabled: page == 1,
					},
					discordgo.Button{
						Label:    "➡️ Next",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("leaderboard_next|%d", page+1),
						Disabled: page == totalPages,
					},
				},
			})
		}

		if existingMessageID := lum.getTrackedMessageID(channelID); existingMessageID != "" {
			editedMessage, err := lum.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				ID:         existingMessageID,
				Channel:    channelID,
				Embeds:     &[]*discordgo.MessageEmbed{embed},
				Components: &components,
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
				Components: &components,
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
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
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
