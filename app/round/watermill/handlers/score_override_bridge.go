package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

const (
	metaDiscordMessageID = "discord_message_id"
	metaChannelID        = "channel_id"
)

// HandleScoreOverrideSuccess bridges CorrectScore success events into the round Discord update flow
// by publishing a RoundParticipantScoreUpdated event so the embed refresh logic is reused.
func (h *RoundHandlers) HandleScoreOverrideSuccess(ctx context.Context, payload *sharedevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Pull discord message id & channel id (if any) from context. Channel may be absent.
	messageID, ok := ctx.Value(metaDiscordMessageID).(string)
	if !ok {
		messageID = ""
	}
	channelID, ok := ctx.Value(metaChannelID).(string)
	if !ok {
		channelID = "" // optional; Update handler falls back to config if empty
	}

	participantPayload := &roundevents.ParticipantScoreUpdatedPayloadV1{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		UserID:         payload.UserID,
		Score:          payload.Score,
		ChannelID:      channelID,
		EventMessageID: messageID,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundParticipantScoreUpdatedV1,
			Payload: participantPayload,
			Metadata: map[string]string{
				"discord_message_id": messageID,
			},
		},
	}, nil
}
