package finalizeround

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// FinalizeScorecardEmbed updates the round embed when a round is finalized
func (frm *finalizeRoundManager) FinalizeScorecardEmbed(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayload) (FinalizeRoundOperationResult, error) {
	return frm.operationWrapper(ctx, "FinalizeScorecardEmbed", func(ctx context.Context) (FinalizeRoundOperationResult, error) {
		// Validate input arguments
		if frm.session == nil {
			err := fmt.Errorf("discord session is nil")
			frm.logger.ErrorContext(ctx, "Discord session is nil in FinalizeScorecardEmbed")
			return FinalizeRoundOperationResult{Error: err}, err // Return both result and error
		}

		// Check for empty or nil UUID string
		if eventMessageID == "" || channelID == "" || eventMessageID == uuid.Nil.String() {
			err := fmt.Errorf("missing channel or message ID for finalization update")
			frm.logger.ErrorContext(ctx, "Missing channel or message ID for finalization update")
			return FinalizeRoundOperationResult{Error: err}, err // Return both result and error
		}

		// Transform the round payload data into a Discord embed and components
		// Assumed this method exists and does the transformation using the embedPayload
		embed, components, err := frm.TransformRoundToFinalizedScorecard(embedPayload)
		if err != nil {
			frm.logger.ErrorContext(ctx, "Failed to transform round to finalized scorecard embed data",
				attr.Error(err),
				attr.RoundID("round_id", embedPayload.RoundID),    // Assuming RoundID is in payload and attr helper supports it
				attr.String("discord_message_id", eventMessageID), // Log message ID for context
				attr.String("channel_id", channelID),              // Log channel ID for context
			)
			return FinalizeRoundOperationResult{Error: fmt.Errorf("failed to prepare embed data: %w", err)}, fmt.Errorf("failed to prepare embed data: %w", err) // Return both result and error
		}

		// Ensure embed and components are not nil before using them
		if embed == nil {
			err := fmt.Errorf("transformed embed is nil")
			frm.logger.ErrorContext(ctx, "Transformed embed is nil",
				attr.RoundID("round_id", embedPayload.RoundID),
				attr.String("discord_message_id", eventMessageID),
				attr.String("channel_id", channelID),
			)
			return FinalizeRoundOperationResult{Error: err}, err // Return both result and error
		}

		// Create the MessageEdit struct to update the Discord message using the provided IDs
		edit := &discordgo.MessageEdit{
			Channel:    channelID,                         // Use the provided channel ID
			ID:         eventMessageID,                    // Use the provided message ID
			Embeds:     &[]*discordgo.MessageEmbed{embed}, // Use pointer to slice
			Components: &components,                       // Use pointer to slice
		}

		// Edit the Discord message via the session
		updatedMsg, err := frm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			wrappedErr := fmt.Errorf("failed to edit embed for finalization: %w", err)
			frm.logger.ErrorContext(ctx, "Failed to update embed for finalization",
				attr.Error(wrappedErr),
				attr.String("discord_message_id", eventMessageID),
				attr.String("channel_id", channelID),
				attr.RoundID("round_id", embedPayload.RoundID), // Log RoundID for context
			)
			return FinalizeRoundOperationResult{Error: wrappedErr}, wrappedErr // Return both result and error
		}

		// Log successful embed update
		frm.logger.InfoContext(ctx, "Successfully finalized round embed on Discord",
			attr.String("discord_message_id", eventMessageID),
			attr.String("channel_id", channelID),
			attr.RoundID("round_id", embedPayload.RoundID), // Log RoundID for context
		)

		// Return success result with the updated message info if needed
		return FinalizeRoundOperationResult{Success: updatedMsg}, nil // Return success result
	})
}
