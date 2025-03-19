package scoreround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

// SendScoreUpdateConfirmation sends a confirmation message when a score is successfully updated
func (srm *scoreRoundManager) SendScoreUpdateConfirmation(channelID string, userID roundtypes.UserID, score *int) error {

	ctx := context.Background() // Consider passing a context from the caller

	srm.logger.Info(ctx, "Sending score update confirmation",
		attr.String("channel_id", channelID),
		attr.String("user_id", string(userID)),
		attr.Int("score", *score))

	_, err := srm.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s> Your score of %d has been recorded!", string(userID), *score),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Users: []string{string(userID)},
		},
	})

	if err != nil {
		srm.logger.Error(ctx, "Failed to send score update confirmation",
			attr.Error(err),
			attr.String("channel_id", channelID),
			attr.String("user_id", string(userID)))
		return err
	}

	return nil
}

// SendScoreUpdateError sends an error message when a score update fails
func (srm *scoreRoundManager) SendScoreUpdateError(userID roundtypes.UserID, errorMsg string) error {
	ctx := context.Background() // Consider passing a context from the caller

	srm.logger.Info(ctx, "Sending score update error notification",
		attr.String("user_id", string(userID)),
		attr.String("error", errorMsg))

	// Find a DM channel with the user
	dmChannel, err := srm.session.UserChannelCreate(string(userID))
	if err != nil {
		srm.logger.Error(ctx, "Failed to create DM channel",
			attr.Error(err),
			attr.String("user_id", string(userID)))
		return err
	}

	_, err = srm.session.ChannelMessageSend(dmChannel.ID,
		fmt.Sprintf("We encountered an error updating your score: %s", errorMsg))

	if err != nil {
		srm.logger.Error(ctx, "Failed to send score update error message",
			attr.Error(err),
			attr.String("user_id", string(userID)))
		return err
	}

	return nil
}
