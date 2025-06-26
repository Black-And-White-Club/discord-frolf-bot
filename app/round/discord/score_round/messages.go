package scoreround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// SendScoreUpdateConfirmation sends a confirmation message when a score is successfully updated
func (srm *scoreRoundManager) SendScoreUpdateConfirmation(ctx context.Context, channelID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_score_update_confirmation")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(userID))

	srm.logger.InfoContext(ctx, "Sending score update confirmation",
		attr.String("channel_id", channelID),
		attr.String("user_id", string(userID)),
		attr.Any("score", *score))

	return srm.operationWrapper(ctx, "send_score_update_confirmation", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		_, err := srm.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("<@%s> Your score of %d has been recorded!", string(userID), *score),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Users: []string{string(userID)},
			},
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to send score update confirmation",
				attr.Error(err),
				attr.String("channel_id", channelID),
				attr.String("user_id", string(userID)))
			return ScoreRoundOperationResult{Error: err}, err
		}

		return ScoreRoundOperationResult{Success: "Score update confirmation sent successfully"}, nil
	})
}

// SendScoreUpdateError sends an error message when a score update fails
func (srm *scoreRoundManager) SendScoreUpdateError(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_score_update_error")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(userID))

	srm.logger.InfoContext(ctx, "Sending score update error notification",
		attr.String("user_id", string(userID)),
		attr.String("error", errorMsg))

	return srm.operationWrapper(ctx, "send_score_update_error", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		// Find a DM channel with the user
		dmChannel, err := srm.session.UserChannelCreate(string(userID))
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to create DM channel",
				attr.Error(err),
				attr.String("user_id", string(userID)))
			return ScoreRoundOperationResult{Error: err}, err // Return the error directly
		}

		_, err = srm.session.ChannelMessageSend(dmChannel.ID,
			fmt.Sprintf("We encountered an error updating your score: %s", errorMsg))
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to send score update error message",
				attr.Error(err),
				attr.String("user_id", string(userID)))
			return ScoreRoundOperationResult{Error: err}, err // Return the error directly
		}

		return ScoreRoundOperationResult{Success: "Score update error notification sent successfully"}, nil
	})
}
