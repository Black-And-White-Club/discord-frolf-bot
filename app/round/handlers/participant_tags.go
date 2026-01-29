package handlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// frolf-bot-discord/app/round/watermill/handlers/handlers.go

// HandleScheduledRoundsSynced (formerly HandleTagsUpdatedForScheduledRounds)
// This processes the result of the sync and refreshes the Discord embeds.
func (h *RoundHandlers) HandleScheduledRoundsSynced(
	ctx context.Context,
	payload *roundevents.ScheduledRoundsSyncedPayloadV1,
) ([]handlerwrapper.Result, error) {

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

	// Use the tag update manager to update Discord embeds
	// Note: Pass the payload directly; the manager might need to be updated
	// if it strictly expects the old payload type.
	result, err := h.service.GetTagUpdateManager().UpdateDiscordEmbedsWithTagChanges(ctx, *payload, tagUpdates)
	if err != nil {
		return nil, fmt.Errorf("failed to update Discord embeds: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("discord embed update failed: %w", result.Error)
	}

	return nil, nil
}

// HandleRoundParticipantsUpdated processes round participant updates and updates Discord embeds
func (h *RoundHandlers) HandleRoundParticipantsUpdated(ctx context.Context, payload *roundevents.RoundParticipantsUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Get guild config to find the event channel ID
	guildConfig, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
	if err != nil {
		h.logger.WarnContext(ctx, "Failed to get guild config for round participants update",
			attr.String("guild_id", string(payload.GuildID)),
			attr.Error(err))
		return []handlerwrapper.Result{}, nil
	}
	if guildConfig == nil || guildConfig.EventChannelID == "" {
		h.logger.WarnContext(ctx, "Missing event channel ID for round participants update",
			attr.String("guild_id", string(payload.GuildID)))
		return []handlerwrapper.Result{}, nil
	}

	// Categorize participants by response
	accepted := []roundtypes.Participant{}
	declined := []roundtypes.Participant{}
	tentative := []roundtypes.Participant{}

	for _, participant := range payload.Round.Participants {
		switch participant.Response {
		case roundtypes.ResponseAccept:
			accepted = append(accepted, participant)
		case roundtypes.ResponseDecline:
			declined = append(declined, participant)
		case roundtypes.ResponseTentative:
			tentative = append(tentative, participant)
		}
	}

	// Update the Discord embed using the RoundRsvpManager
	result, err := h.service.GetRoundRsvpManager().UpdateRoundEventEmbed(ctx, guildConfig.EventChannelID, payload.Round.EventMessageID, accepted, declined, tentative)
	if err != nil {
		return nil, fmt.Errorf("failed to update Discord embed for round %s: %w", payload.RoundID, err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("discord embed update failed for round %s: %w", payload.RoundID, result.Error)
	}

	return nil, nil // No further messages to publish
}
