package leaderboardupdated

import (
	"context"
	"fmt"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

type LeaderboardEntry struct {
	Rank   sharedtypes.TagNumber `json:"rank"`
	UserID sharedtypes.DiscordID `json:"user_id"`
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
		end := start + entriesPerPage
		if end > int32(len(leaderboard)) {
			end = int32(len(leaderboard))
		}

		// Build leaderboard fields
		fields := []*discordgo.MessageEmbedField{}
		for _, entry := range leaderboard[start:end] {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("#%d", entry.Rank),
				Value:  fmt.Sprintf("<@%s>", entry.UserID),
				Inline: false,
			})
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🏆 Leaderboard",
			Description: fmt.Sprintf("Page %d/%d", page, totalPages),
			Color:       0xFFD700, // Gold color
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Updated: %s", time.Now().Format(time.RFC1123)),
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

		message, err := lum.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		})
		if err != nil {
			err := fmt.Errorf("failed to send leaderboard message: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, err
		}

		return LeaderboardUpdateOperationResult{Success: message}, nil
	})
}
