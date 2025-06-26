package tagupdates

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func (tum *tagUpdateManager) UpdateDiscordEmbedsWithTagChanges(ctx context.Context, payload roundevents.TagsUpdatedForScheduledRoundsPayload, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (TagUpdateOperationResult, error) {
	return tum.operationWrapper(ctx, "UpdateDiscordEmbedsWithTagChanges", func(ctx context.Context) (TagUpdateOperationResult, error) {
		tum.logger.InfoContext(ctx, "Processing Discord embed updates for tag changes",
			attr.Int("rounds_to_update", len(payload.UpdatedRounds)),
			attr.Int("tag_updates", len(tagUpdates)))

		successfulUpdates := 0
		failedUpdates := 0

		// Process each round that needs Discord embed updating
		for _, roundInfo := range payload.UpdatedRounds {
			tum.logger.InfoContext(ctx, "Updating Discord embed for round",
				attr.String("round_id", roundInfo.RoundID.String()),
				attr.String("event_message_id", roundInfo.EventMessageID),
				attr.String("title", string(roundInfo.Title)))

			// Extract channel ID from config - this may need to be added to RoundUpdateInfo
			// For now, we'll use the channel from config or try to determine it
			channelID := tum.config.Discord.EventChannelID

			result, err := tum.UpdateTagsInEmbed(ctx, channelID, roundInfo.EventMessageID, tagUpdates)
			if err != nil || result.Error != nil {
				tum.logger.ErrorContext(ctx, "Failed to update Discord embed",
					attr.String("round_id", roundInfo.RoundID.String()),
					attr.String("event_message_id", roundInfo.EventMessageID),
					attr.Error(err))
				failedUpdates++
				continue
			}

			tum.logger.InfoContext(ctx, "Successfully updated Discord embed",
				attr.String("round_id", roundInfo.RoundID.String()),
				attr.String("event_message_id", roundInfo.EventMessageID))
			successfulUpdates++
		}

		tum.logger.InfoContext(ctx, "Completed Discord embed updates",
			attr.Int("successful_updates", successfulUpdates),
			attr.Int("failed_updates", failedUpdates))

		return TagUpdateOperationResult{Success: "Discord embeds updated"}, nil
	})
}
