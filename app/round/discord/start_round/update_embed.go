package startround

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// UpdateRoundToScorecard updates a round event embed to display the scorecard format
func (srm *startRoundManager) UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayload) error {

	// Transform the embed to scorecard format
	embed, components, err := srm.TransformRoundToScorecard(payload)
	if err != nil {
		srm.logger.Error(ctx, "Failed to transform round to scorecard", attr.Error(err))
		return fmt.Errorf("failed to transform round to scorecard: %w", err)
	}

	// Update the message with new embed and components
	_, err = srm.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    channelID,
		ID:         messageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		srm.logger.Error(ctx, "Failed to update round embed to scorecard",
			attr.Error(err),
			attr.String("message_id", messageID))
		return fmt.Errorf("failed to update round embed: %w", err)
	}

	srm.logger.Info(ctx, "Successfully updated round embed to scorecard",
		attr.String("message_id", messageID),
		attr.String("channel_id", channelID))
	return nil
}
