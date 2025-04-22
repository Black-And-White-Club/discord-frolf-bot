package deleteround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// DeleteRoundEventEmbed removes the message with the given messageID from the specified channel.
func (drm *deleteRoundManager) DeleteRoundEventEmbed(ctx context.Context, eventMessageID sharedtypes.RoundID, channelID string) (DeleteRoundOperationResult, error) {
	return drm.operationWrapper(ctx, "DeleteRoundEventEmbed", func(ctx context.Context) (DeleteRoundOperationResult, error) {
		drm.logger.InfoContext(ctx, "Attempting to delete round embed",
			attr.String("channel_id", channelID),
			attr.RoundID("message_id", eventMessageID))

		// Ensure both channelID and eventMessageID are provided
		if channelID == "" || eventMessageID == sharedtypes.RoundID(uuid.Nil) {
			err := fmt.Errorf("channelID or eventMessageID is missing")
			drm.logger.ErrorContext(ctx, "Missing channelID or eventMessageID",
				attr.String("channel_id", channelID),
				attr.RoundID("message_id", eventMessageID),
				attr.Error(err))
			return DeleteRoundOperationResult{Error: err, Success: false}, nil // Explicitly set Success to false
		}

		// Attempt to delete the message
		err := drm.session.ChannelMessageDelete(channelID, eventMessageID.String())
		if err != nil {
			wrappedErr := fmt.Errorf("failed to delete message: %w", err)
			drm.logger.ErrorContext(ctx, "Failed to delete round embed",
				attr.String("channel_id", channelID),
				attr.RoundID("message_id", eventMessageID),
				attr.Error(wrappedErr))
			return DeleteRoundOperationResult{Error: wrappedErr, Success: false}, nil // Explicitly set Success to false
		}

		drm.logger.InfoContext(ctx, "Successfully deleted round embed",
			attr.String("channel_id", channelID),
			attr.RoundID("message_id", eventMessageID))

		return DeleteRoundOperationResult{Success: true}, nil
	})
}
