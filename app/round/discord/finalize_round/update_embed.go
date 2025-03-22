package finalizeround

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// FinalizeScorecardEmbed updates the round embed when a round is finalized
func (frm *finalizeRoundManager) FinalizeScorecardEmbed(ctx context.Context, eventMessageID, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.Message, error) {
	if frm.session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	embed, components, err := frm.TransformRoundToFinalizedScorecard(embedPayload)
	if err != nil {
		frm.logger.Error(ctx, "Failed to transform round to finalized scorecard",
			attr.Error(err),
			attr.String("round_id", fmt.Sprintf("%d", embedPayload.RoundID)))
		return nil, err
	}

	if eventMessageID == "" || channelID == "" {
		return nil, fmt.Errorf("missing channel or message ID for finalization update")
	}

	edit := &discordgo.MessageEdit{
		Channel:    channelID,
		ID:         eventMessageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}

	updatedMsg, err := frm.session.ChannelMessageEditComplex(edit)
	if err != nil {
		frm.logger.Error(ctx, "Failed to update embed for finalization",
			attr.Error(err),
			attr.String("message_id", eventMessageID),
			attr.String("channel_id", channelID))
		return nil, err
	}

	frm.logger.Info(ctx, "Successfully finalized round in embed",
		attr.String("message_id", eventMessageID),
		attr.String("channel_id", channelID))

	return updatedMsg, nil
}
