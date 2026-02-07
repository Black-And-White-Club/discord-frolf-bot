package handlers

import (
	"context"
	"fmt"
	"strings"

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
		// Fallback: try to look up message ID from the map if metadata is missing
		if msgID, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			discordMessageID = msgID
		} else {
			return nil, fmt.Errorf("discord_message_id not found or is empty in message metadata for round %s", payload.RoundID.String())
		}
	}

	result, err := h.service.GetDeleteRoundManager().DeleteRoundEventEmbed(ctx, discordMessageID, h.config.GetEventChannelID())
	if err != nil {
		return nil, fmt.Errorf("error calling delete round embed service: %w", err)
	}

	_, ok = result.Success.(bool)
	if !ok {
		return nil, fmt.Errorf("unexpected type for result.Success in DeleteRoundEventEmbed result for round %s", payload.RoundID.String())
	}

	// Delete the native Discord Scheduled Event (best-effort).
	if payload.DiscordEventID != "" {
		session := h.service.GetSession()
		if err := session.GuildScheduledEventDelete(string(payload.GuildID), payload.DiscordEventID); err != nil {
			// Ignore 404 (Unknown Guild Scheduled Event) as it likely means the user already deleted it
			if !strings.Contains(err.Error(), "10070") { // 10070 = Unknown Guild Scheduled Event
				h.logger.WarnContext(ctx, "failed to delete native event",
					"discord_event_id", payload.DiscordEventID,
					"error", err,
				)
			}
		}
		// Clean up the NativeEventMap entry.
		if nativeEventMap := h.service.GetNativeEventMap(); nativeEventMap != nil {
			nativeEventMap.Delete(payload.RoundID)
		}
	}

	// Avoid returning trace events here to prevent publish failures from
	// causing the handler to be retried (which would duplicate embed
	// deletion attempts). An empty result list acknowledges successful
	// handling.
	return []handlerwrapper.Result{}, nil
}
