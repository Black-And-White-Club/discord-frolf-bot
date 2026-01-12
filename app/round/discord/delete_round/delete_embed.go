package deleteround

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// DeleteRoundEventEmbed removes the message with the given messageID from the specified channel.
func (drm *deleteRoundManager) DeleteRoundEventEmbed(ctx context.Context, discordMessageID string, channelID string) (DeleteRoundOperationResult, error) {
	return drm.operationWrapper(ctx, "DeleteRoundEventEmbed", func(ctx context.Context) (DeleteRoundOperationResult, error) {
		// If channelID is empty, try to resolve from guild config if possible
		resolvedChannelID := channelID
		guildID, _ := ctx.Value("guild_id").(string) // You may want to use a typed key in your context
		if resolvedChannelID == "" && drm.guildConfigResolver != nil && guildID != "" {
			guildConfig, err := drm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
			if err == nil && guildConfig != nil && guildConfig.EventChannelID != "" {
				resolvedChannelID = guildConfig.EventChannelID
				drm.logger.InfoContext(ctx, "Resolved channel ID from guild config", attr.String("channel_id", resolvedChannelID))
			} else {
				drm.logger.WarnContext(ctx, "Failed to resolve channel ID from guild config, deletion may fail", attr.Error(err))
			}
		}

		drm.logger.InfoContext(ctx, "Attempting to delete round embed via Discord API",
			attr.String("channel_id", resolvedChannelID),
			attr.String("message_id", discordMessageID))

		// Ensure both channelID and discordMessageID are provided (check for empty string)
		if resolvedChannelID == "" || discordMessageID == "" {
			err := fmt.Errorf("channelID or discordMessageID is missing")
			drm.logger.ErrorContext(ctx, "Missing channelID or discordMessageID for deletion",
				attr.String("channel_id", resolvedChannelID),
				attr.String("message_id", discordMessageID),
				attr.Error(err))
			return DeleteRoundOperationResult{Error: err, Success: false}, nil
		}

		err := drm.session.ChannelMessageDelete(resolvedChannelID, discordMessageID)
		if err != nil {
			// Provide more context in the wrapped error
			wrappedErr := fmt.Errorf("failed to delete message: %w", err)
			drm.logger.ErrorContext(ctx, "Discord API call failed to delete message",
				attr.String("channel_id", resolvedChannelID),
				attr.String("message_id", discordMessageID),
				attr.Error(wrappedErr))
			return DeleteRoundOperationResult{Error: wrappedErr, Success: false}, nil
		}

		drm.logger.InfoContext(ctx, "Successfully sent Discord API delete command for message",
			attr.String("channel_id", resolvedChannelID),
			attr.String("message_id", discordMessageID))

		return DeleteRoundOperationResult{Success: true}, nil
	})
}
