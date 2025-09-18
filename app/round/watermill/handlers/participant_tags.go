package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleTagsUpdatedForScheduledRounds(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleTagsUpdatedForScheduledRounds",
		&roundevents.TagsUpdatedForScheduledRoundsPayload{}, // Correct payload type
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagsUpdatedPayload := payload.(*roundevents.TagsUpdatedForScheduledRoundsPayload)

			h.Logger.InfoContext(ctx, "Received TagsUpdatedForScheduledRounds event",
				attr.CorrelationIDFromMsg(msg),
				attr.Int("rounds_to_update", len(tagsUpdatedPayload.UpdatedRounds)),
				attr.Int("total_participants_updated", tagsUpdatedPayload.Summary.ParticipantsUpdated),
			)

			if len(tagsUpdatedPayload.UpdatedRounds) == 0 {
				h.Logger.InfoContext(ctx, "No rounds to update, skipping Discord updates")
				return nil, nil
			}

			// Attach guild_id of first round (all rounds share guild) to context for downstream resolution
			if len(tagsUpdatedPayload.UpdatedRounds) > 0 {
				ctx = context.WithValue(ctx, "guild_id", string(tagsUpdatedPayload.UpdatedRounds[0].GuildID))
				h.Logger.DebugContext(ctx, "Attached guild_id to context for tag update operation",
					attr.String("guild_id", string(tagsUpdatedPayload.UpdatedRounds[0].GuildID)),
				)
			}

			// Extract tag changes from the updated participants
			tagUpdates := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)
			for _, roundInfo := range tagsUpdatedPayload.UpdatedRounds {
				for _, participant := range roundInfo.UpdatedParticipants {
					if participant.TagNumber != nil {
						tagUpdates[participant.UserID] = participant.TagNumber
					}
				}
			}

			h.Logger.InfoContext(ctx, "Extracted tag updates for Discord",
				attr.Int("total_tag_updates", len(tagUpdates)))

			// Use the tag update manager to update Discord embeds
			result, err := h.RoundDiscord.GetTagUpdateManager().UpdateDiscordEmbedsWithTagChanges(ctx, *tagsUpdatedPayload, tagUpdates)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update Discord embeds",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to update Discord embeds: %w", err)
			}

			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Discord embed update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(result.Error),
				)
				return nil, result.Error
			}

			h.Logger.InfoContext(ctx, "Successfully updated Discord embeds for tag changes",
				attr.CorrelationIDFromMsg(msg),
				attr.Int("embeds_updated", len(tagsUpdatedPayload.UpdatedRounds)),
			)

			return nil, nil // No further messages to publish
		},
	)(msg)
}
