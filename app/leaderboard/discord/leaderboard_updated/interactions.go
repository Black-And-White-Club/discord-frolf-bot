package leaderboardupdated

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (lum *leaderboardUpdateManager) HandleLeaderboardPagination(ctx context.Context, i *discordgo.InteractionCreate) (LeaderboardUpdateOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_leaderboard_pagination")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	lum.logger.InfoContext(ctx, "Handling leaderboard pagination interaction",
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID),
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)),
	)

	return lum.operationWrapper(ctx, "handle_leaderboard_pagination", func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
		customIDParts := strings.Split(i.MessageComponentData().CustomID, "|")
		if len(customIDParts) != 2 {
			err := fmt.Errorf("invalid CustomID format: %s", i.MessageComponentData().CustomID)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, nil
		}

		newPage, err := strconv.Atoi(customIDParts[1])
		if err != nil {
			err := fmt.Errorf("error parsing page number: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, nil
		}

		if len(i.Message.Embeds) == 0 {
			err := fmt.Errorf("no embeds found in message")
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, nil
		}
		embed := i.Message.Embeds[0]

		var currentPage, totalPages int
		if _, err := fmt.Sscanf(embed.Description, "Page %d/%d", &currentPage, &totalPages); err != nil {
			err := fmt.Errorf("error parsing embed page numbers: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, nil
		}

		if newPage < 1 || newPage > totalPages {
			err := fmt.Errorf("requested page out of bounds: %d", newPage)
			lum.logger.WarnContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Failure: "page out of range"}, nil
		}

		const entriesPerPage = 10
		start := (newPage - 1) * entriesPerPage
		end := start + entriesPerPage
		if end > len(embed.Fields) {
			end = len(embed.Fields)
		}

		newEmbed := *embed
		newEmbed.Description = fmt.Sprintf("Page %d/%d", newPage, totalPages)
		newEmbed.Fields = embed.Fields[start:end]

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

		err = lum.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{&newEmbed},
				Components: components,
			},
		})
		if err != nil {
			err := fmt.Errorf("error updating leaderboard message: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, err
		}

		return LeaderboardUpdateOperationResult{Success: "pagination updated"}, nil
	})
}
