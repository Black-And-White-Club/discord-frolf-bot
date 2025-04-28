package startround

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// UpdateRoundToScorecard uses the operation wrapper to update a round event embed.
func (m *startRoundManager) UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayload) (StartRoundOperationResult, error) {
	return m.operationWrapper(ctx, "UpdateRoundToScorecard", func(ctx context.Context) (StartRoundOperationResult, error) {
		// Validate inputs before proceeding
		if channelID == "" {
			err := errors.New("missing channel ID")
			m.logger.ErrorContext(ctx, "Missing channel ID for UpdateRoundToScorecard")
			return StartRoundOperationResult{Error: err}, nil
		}
		if messageID == "" {
			err := errors.New("missing message ID")
			m.logger.ErrorContext(ctx, "Missing message ID for UpdateRoundToScorecard")
			return StartRoundOperationResult{Error: err}, nil
		}

		transformResult, err := m.TransformRoundToScorecard(ctx, payload)
		if err != nil {
			// This error indicates a problem with the wrapper itself, not the transformation logic
			m.logger.ErrorContext(ctx, "Failed to run TransformRoundToScorecard wrapper", attr.Error(err))
			return StartRoundOperationResult{}, fmt.Errorf("failed to run TransformRoundToScorecard wrapper: %w", err)
		}

		// Check for an error within the result of TransformRoundToScorecard
		if transformResult.Error != nil {
			m.logger.ErrorContext(ctx, "TransformRoundToScorecard returned error in result", attr.Error(transformResult.Error))
			// Return the error from the transformation in the result struct
			return StartRoundOperationResult{Error: fmt.Errorf("transformation failed: %w", transformResult.Error)}, nil
		}

		transformedData, ok := transformResult.Success.(struct {
			Embed      *discordgo.MessageEmbed
			Components []discordgo.MessageComponent
		})
		if !ok {
			// Handle unexpected type from TransformRoundToScorecard success
			err := errors.New("unexpected success type from TransformRoundToScorecard")
			m.logger.ErrorContext(ctx, "Unexpected type from TransformRoundToScorecard success", attr.Error(err))
			return StartRoundOperationResult{Error: err}, nil
		}

		embed := transformedData.Embed
		components := transformedData.Components

		// Update the message with new embed and components
		_, err = m.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    channelID,
			ID:         messageID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			// If updating the message failed, return an error in the result struct.
			m.logger.ErrorContext(ctx, "Failed to update round embed to scorecard",
				attr.Error(err),
				attr.String("message_id", messageID))
			return StartRoundOperationResult{Error: fmt.Errorf("failed to update round embed: %w", err)}, nil
		}

		m.logger.InfoContext(ctx, "Successfully updated round embed to scorecard",
			attr.String("message_id", messageID),
			attr.String("channel_id", channelID))

		// Return a successful result
		return StartRoundOperationResult{Success: true}, nil
	})
}
