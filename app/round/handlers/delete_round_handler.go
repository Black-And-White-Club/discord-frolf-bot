package handlers

import (
	"context"
	"fmt"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundDeleteRequested handles the RoundDeleteRequestDiscordV1 event
// and publishes the domain event RoundDeleteRequestedV1.
func (h *RoundHandlers) HandleRoundDeleteRequested(ctx context.Context, payload *discordroundevents.RoundDeleteRequestDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	// Map to domain payload
	domainPayload := roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              sharedtypes.GuildID(payload.GuildID),
		RoundID:              payload.RoundID,
		RequestingUserUserID: payload.UserID,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundDeleteRequestedV1,
			Payload: domainPayload,
		},
	}, nil
}

// HandleRoundDeleted handles the RoundDeleted event using the standardized wrapper
func (h *RoundHandlers) HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error) {
	// Get message ID from context (set by wrapper from message metadata)
	discordMessageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || discordMessageID == "" {
		return nil, fmt.Errorf("discord_message_id not found or is empty in message metadata for round %s", payload.RoundID.String())
	}

	result, err := h.service.GetDeleteRoundManager().DeleteRoundEventEmbed(ctx, discordMessageID, h.config.GetEventChannelID())
	if err != nil {
		return nil, fmt.Errorf("error calling delete round embed service: %w", err)
	}

	_, ok = result.Success.(bool)
	if !ok {
		return nil, fmt.Errorf("unexpected type for result.Success in DeleteRoundEventEmbed result for round %s", payload.RoundID.String())
	}

	// Avoid returning trace events here to prevent publish failures from
	// causing the handler to be retried (which would duplicate embed
	// deletion attempts). An empty result list acknowledges successful
	// handling.
	return []handlerwrapper.Result{}, nil
}
