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

		// Determine the channel from the message
		channelID := i.ChannelID

		// Pull the cached leaderboard data for this channel
		leaderboard := lum.getCachedLeaderboard(channelID)
		if len(leaderboard) == 0 {
			lum.logger.WarnContext(ctx, "No cached leaderboard data found for pagination, bot may have restarted",
				attr.String("channel_id", channelID),
			)
			// Gracefully inform the user — ephemeral so it doesn't clutter the channel
			err := lum.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "⏳ The leaderboard data is being refreshed. Please wait a moment and try again, or use `/leaderboard` to reload.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				lum.logger.ErrorContext(ctx, "Failed to send cache-miss response", attr.Error(err))
			}
			return LeaderboardUpdateOperationResult{Failure: "no cached leaderboard data"}, nil
		}

		embed, components := buildLeaderboardEmbed(leaderboard, int32(newPage))

		err = lum.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
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
