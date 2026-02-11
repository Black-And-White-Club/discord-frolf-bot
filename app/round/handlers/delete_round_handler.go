package handlers

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
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
	// 1. Get IDs from context (set by wrapper from message metadata)
	discordMessageID, _ := ctx.Value("discord_message_id").(string)
	channelID, _ := ctx.Value("channel_id").(string)

	// 2. Fallback to payload data if metadata is missing (ensure protocol consistency)
	if discordMessageID == "" {
		discordMessageID = payload.EventMessageID
	}
	if channelID == "" {
		channelID = payload.ChannelID
	}

	// 3. Last-resort fallback to message map lookup for discordMessageID
	if discordMessageID == "" {
		if msgID, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			discordMessageID = msgID
		}
	}

	// 4. Last-resort fallback to config for channelID
	if channelID == "" {
		if h.guildConfigResolver != nil {
			guildCfg, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
			if err != nil || guildCfg == nil {
				h.logger.WarnContext(ctx, "failed to resolve guild config for round deletion, falling back to global config",
					attr.String("guild_id", string(payload.GuildID)),
					attr.Error(err))
				channelID = h.config.GetEventChannelID()
			} else {
				channelID = guildCfg.EventChannelID
			}
		} else {
			channelID = h.config.GetEventChannelID()
		}
	}

	// 5. Final validation before call
	if discordMessageID == "" {
		return nil, fmt.Errorf("discord_message_id not found or is empty in message metadata or payload for round %s", payload.RoundID.String())
	}

	result, err := h.service.GetDeleteRoundManager().DeleteRoundEventEmbed(ctx, discordMessageID, channelID)
	if err != nil {
		return nil, fmt.Errorf("error calling delete round embed service: %w", err)
	}

	// The deleteround service returns Success as a bool interface{}
	success, ok := result.Success.(bool)
	if !ok || !success {
		// If Success is not true, check if there's an error in the result
		if result.Error != nil {
			return nil, fmt.Errorf("deleteround service returned error for round %s: %w", payload.RoundID.String(), result.Error)
		}
		return nil, fmt.Errorf("deleteround service reported failure for round %s (success=false)", payload.RoundID.String())
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
