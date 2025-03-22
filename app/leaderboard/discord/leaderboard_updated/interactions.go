package leaderboardupdated

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (lbm *leaderboardUpdateManager) HandleLeaderboardPagination(ctx context.Context, i *discordgo.InteractionCreate) {
	// Extract page number from button CustomID
	customIDParts := strings.Split(i.MessageComponentData().CustomID, "|")
	if len(customIDParts) != 2 {
		slog.Error("Invalid CustomID format", attr.String("custom_id", i.MessageComponentData().CustomID))
		return
	}

	newPage, err := strconv.Atoi(customIDParts[1]) // Extract page number
	if err != nil {
		slog.Error("Error parsing page number", attr.Error(err))
		return
	}

	// Get the original embed from the message
	if len(i.Message.Embeds) == 0 {
		slog.Error("No embeds found in message.")
		return
	}
	embed := i.Message.Embeds[0]

	// Extract current page & total pages from embed description
	var currentPage, totalPages int
	if _, err := fmt.Sscanf(embed.Description, "Page %d/%d", &currentPage, &totalPages); err != nil {
		slog.Error("Error parsing embed page", attr.Error(err))
		return
	}

	// Ensure the new page is within valid range
	if newPage < 1 || newPage > totalPages {
		return
	}

	// Define entries per page (same as SendLeaderboardEmbed)
	const entriesPerPage = 10
	start := (newPage - 1) * entriesPerPage
	end := start + entriesPerPage
	if end > len(embed.Fields) {
		end = len(embed.Fields)
	}

	// Update embed fields for the correct page
	newEmbed := *embed // Copy original embed
	newEmbed.Description = fmt.Sprintf("Page %d/%d", newPage, totalPages)
	newEmbed.Fields = embed.Fields[start:end]

	// Update buttons for the new page
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "⬅️ Previous",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("leaderboard_prev|%d", newPage-1),
					Disabled: newPage == 1,
				},
				discordgo.Button{
					Label:    "➡️ Next",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("leaderboard_next|%d", newPage+1),
					Disabled: newPage == totalPages,
				},
			},
		},
	}

	// Acknowledge interaction & edit the message
	err = lbm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage, // Updates the same message
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{&newEmbed},
			Components: components,
		},
	})
	if err != nil {
		slog.Error("Error updating leaderboard message", attr.Error(err))
	}
}
