package finalizeround

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// FinalizeScorecardEmbed updates the round embed when a round is finalized
func (frm *finalizeRoundManager) FinalizeScorecardEmbed(ctx context.Context, eventMessageID, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayload) (FinalizeRoundOperationResult, error) {
	return frm.operationWrapper(ctx, "FinalizeScorecardEmbed", func(ctx context.Context) (FinalizeRoundOperationResult, error) {
		if frm.session == nil {
			err := fmt.Errorf("session is nil")
			return FinalizeRoundOperationResult{Error: err}, err
		}

		if eventMessageID == "" || channelID == "" {
			err := fmt.Errorf("missing channel or message ID for finalization update")
			return FinalizeRoundOperationResult{Error: err}, err
		}

		embed, components, err := frm.TransformRoundToFinalizedScorecard(embedPayload)
		if err != nil {
			frm.logger.ErrorContext(ctx, "Failed to transform round to finalized scorecard",
				attr.Error(err),
				attr.String("round_id", fmt.Sprintf("%d", embedPayload.RoundID)))
			return FinalizeRoundOperationResult{Error: err}, nil
		}

		edit := &discordgo.MessageEdit{
			Channel:    channelID,
			ID:         eventMessageID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}

		updatedMsg, err := frm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			wrappedErr := fmt.Errorf("failed to update embed for finalization: %w", err)
			frm.logger.ErrorContext(ctx, "Failed to update embed for finalization",
				attr.Error(wrappedErr),
				attr.String("message_id", eventMessageID),
				attr.String("channel_id", channelID))
			return FinalizeRoundOperationResult{Error: wrappedErr}, wrappedErr
		}

		frm.logger.InfoContext(ctx, "Successfully finalized round in embed",
			attr.String("message_id", eventMessageID),
			attr.String("channel_id", channelID))

		return FinalizeRoundOperationResult{Success: updatedMsg}, nil
	})
}
