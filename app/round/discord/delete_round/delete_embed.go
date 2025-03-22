package deleteround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// DeleteEmbed removes the message with the given messageID from the specified channel.
func (dem *deleteRoundManager) DeleteEmbed(ctx context.Context, eventMessageID roundtypes.EventMessageID, channelID string) (bool, error) {
	dem.logger.Info(ctx, "Attempting to delete round embed",
		attr.String("channel_id", channelID),
		attr.String("message_id", string(eventMessageID)))

	// Ensure both channelID and eventMessageID are provided
	if channelID == "" || eventMessageID == "" {
		dem.logger.Error(ctx, "Missing channelID or eventMessageID",
			attr.String("channel_id", channelID),
			attr.String("message_id", string(eventMessageID)))
		return false, fmt.Errorf("channelID or eventMessageID is missing")
	}

	// Attempt to delete the message
	err := dem.session.ChannelMessageDelete(channelID, string(eventMessageID))
	if err != nil {
		dem.logger.Error(ctx, "Failed to delete round embed",
			attr.String("channel_id", channelID),
			attr.String("message_id", string(eventMessageID)),
			attr.Error(err))
		return false, fmt.Errorf("failed to delete message: %w", err)
	}

	dem.logger.Info(ctx, "Successfully deleted round embed",
		attr.String("channel_id", channelID),
		attr.String("message_id", string(eventMessageID)))

	return true, nil
}
