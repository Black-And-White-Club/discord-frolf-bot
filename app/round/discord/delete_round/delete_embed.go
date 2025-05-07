package deleteround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// DeleteRoundEventEmbed removes the message with the given messageID from the specified channel.
func (drm *deleteRoundManager) DeleteRoundEventEmbed(ctx context.Context, discordMessageID string, channelID string) (DeleteRoundOperationResult, error) {
	return drm.operationWrapper(ctx, "DeleteRoundEventEmbed", func(ctx context.Context) (DeleteRoundOperationResult, error) {
		drm.logger.InfoContext(ctx, "Attempting to delete round embed via Discord API",
			attr.String("channel_id", channelID),
			attr.String("discord_message_id", discordMessageID)) // Log the correct ID

		// Ensure both channelID and discordMessageID are provided (check for empty string)
		if channelID == "" || discordMessageID == "" {
			err := fmt.Errorf("channelID or discordMessageID is missing")
			drm.logger.ErrorContext(ctx, "Missing channelID or discordMessageID for deletion",
				attr.String("channel_id", channelID),
				attr.String("discord_message_id", discordMessageID),
				attr.Error(err))
			return DeleteRoundOperationResult{Error: err, Success: false}, nil
		}

		err := drm.session.ChannelMessageDelete(channelID, discordMessageID)
		if err != nil {
			// Provide more context in the wrapped error
			wrappedErr := fmt.Errorf("failed to delete Discord message %s in channel %s: %w", discordMessageID, channelID, err)
			drm.logger.ErrorContext(ctx, "Discord API call failed to delete message",
				attr.String("channel_id", channelID),
				attr.String("discord_message_id", discordMessageID),
				attr.Error(wrappedErr))

			return DeleteRoundOperationResult{Error: wrappedErr, Success: false}, nil
		}

		drm.logger.InfoContext(ctx, "Successfully sent Discord API delete command for message",
			attr.String("channel_id", channelID),
			attr.String("discord_message_id", discordMessageID))

		return DeleteRoundOperationResult{Success: true}, nil
	})
}
