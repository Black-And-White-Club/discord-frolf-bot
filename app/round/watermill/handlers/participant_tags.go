package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

func (h *RoundHandlers) HandleTagsUpdatedForScheduledRounds(ctx context.Context, payload *roundevents.TagsUpdatedForScheduledRoundsPayloadV1) ([]handlerwrapper.Result, error) {
	if len(payload.UpdatedRounds) == 0 {
		return nil, nil
	}

	// Extract tag changes from the updated participants
	tagUpdates := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)
	for _, roundInfo := range payload.UpdatedRounds {
		for _, participant := range roundInfo.UpdatedParticipants {
			if participant.TagNumber != nil {
				tagUpdates[participant.UserID] = participant.TagNumber
			}
		}
	}

	// Use the tag update manager to update Discord embeds (works for both RSVP and scorecard embeds)
	result, err := h.RoundDiscord.GetTagUpdateManager().UpdateDiscordEmbedsWithTagChanges(ctx, *payload, tagUpdates)
	if err != nil {
		return nil, fmt.Errorf("failed to update Discord embeds: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("discord embed update failed: %w", result.Error)
	}

	return nil, nil // No further messages to publish
}
