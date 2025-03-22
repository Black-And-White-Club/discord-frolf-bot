package leaderboardupdated

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type LeaderboardEntry struct {
	Rank   int    `json:"rank"`
	UserID string `json:"user_id"`
}

func (lbm *leaderboardUpdateManager) SendLeaderboardEmbed(channelID string, leaderboard []LeaderboardEntry, page int) (*discordgo.Message, error) {
	const entriesPerPage = 10
	totalPages := (len(leaderboard) + entriesPerPage - 1) / entriesPerPage

	// Ensure page is within valid range
	if page < 1 {
		page = 1
	} else if page > totalPages {
		page = totalPages
	}

	// Calculate slice range
	start := (page - 1) * entriesPerPage
	end := start + entriesPerPage
	if end > len(leaderboard) {
		end = len(leaderboard)
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

	// Create the embed
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
	if totalPages > 1 { // Only show buttons if there's more than one page
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

	// Send the message
	message, err := lbm.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send leaderboard message: %w", err)
	}

	return message, nil
}
