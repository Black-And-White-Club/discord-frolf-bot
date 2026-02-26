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

		if _, err := strconv.Atoi(customIDParts[1]); err != nil {
			err := fmt.Errorf("error parsing page number: %w", err)
			lum.logger.ErrorContext(ctx, err.Error())
			return LeaderboardUpdateOperationResult{Error: err}, nil
		}

		// Leaderboard pagination has been replaced with a single-page description embed.
		// Old embeds with pagination buttons may still exist; respond ephemerally.
		lum.logger.InfoContext(ctx, "Leaderboard pagination button pressed on old embed — pagination no longer used")
		err := lum.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ The leaderboard has been updated and no longer uses pagination. The next leaderboard refresh will replace this embed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			lum.logger.ErrorContext(ctx, "Failed to respond to stale pagination button", attr.Error(err))
		}
		return LeaderboardUpdateOperationResult{Failure: "pagination no longer supported"}, nil
	})
}
