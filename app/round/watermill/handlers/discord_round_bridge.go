package roundhandlers

import (
	"context"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleDiscordRoundScoreUpdate is a thin transport adapter that converts
// Discord-scoped round score updates into the canonical round domain event.
func (h *RoundHandlers) HandleDiscordRoundScoreUpdate(
	ctx context.Context,
	payload *discordroundevents.RoundScoreUpdateRequestDiscordPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, nil
	}

	// Map the Discord payload to the round domain payload. Preserve transport
	// IDs in metadata so downstream handlers can update embeds and correlate traces.
	backendPayload := roundevents.ScoreUpdateRequestPayloadV1{
		GuildID:   sharedtypes.GuildID(payload.GuildID),
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		Score:     &payload.Score,
		ChannelID: payload.ChannelID,
		MessageID: payload.MessageID,
	}

	md := map[string]string{}
	if payload.MessageID != "" {
		md["discord_message_id"] = payload.MessageID
		md["message_id"] = payload.MessageID
	}
	if payload.ChannelID != "" {
		md["channel_id"] = payload.ChannelID
	}
	if payload.GuildID != "" {
		md["guild_id"] = payload.GuildID
	}

	return []handlerwrapper.Result{{
		Topic:    roundevents.RoundScoreUpdateRequestedV1,
		Payload:  backendPayload,
		Metadata: md,
	}}, nil
}
