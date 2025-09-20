package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	metaDiscordMessageID = "discord_message_id"
	metaChannelID        = "channel_id"
)

// HandleScoreOverrideSuccess bridges CorrectScore success events into the round Discord update flow
// by publishing a DiscordParticipantScoreUpdated event so the embed refresh logic is reused.
func (h *RoundHandlers) HandleScoreOverrideSuccess(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScoreOverrideSuccess",
		&scoreevents.ScoreUpdateSuccessPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p, ok := payload.(*scoreevents.ScoreUpdateSuccessPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleScoreOverrideSuccess")
			}

			// Pull discord message id & channel id (if any) from metadata. Channel may be absent.
			messageID := msg.Metadata.Get(metaDiscordMessageID)
			channelID := msg.Metadata.Get(metaChannelID) // optional; Update handler falls back to config if empty

			if messageID == "" {
				h.Logger.WarnContext(ctx, "score override success missing discord_message_id metadata; no embed update will occur")
				// We still continue; UpdateScoreEmbed requires message id so we will publish anyway (maybe downstream fills it)
			}

			participantPayload := &roundevents.ParticipantScoreUpdatedPayload{
				GuildID:        p.GuildID,
				RoundID:        p.RoundID,
				Participant:    p.UserID,
				Score:          p.Score,
				ChannelID:      channelID,
				EventMessageID: messageID,
			}

			bridgeMsg, err := h.Helpers.CreateResultMessage(msg, participantPayload, roundevents.DiscordParticipantScoreUpdated)
			if err != nil {
				return nil, fmt.Errorf("failed to create bridge message: %w", err)
			}
			// Ensure topic metadata set
			bridgeMsg.Metadata.Set("topic", roundevents.DiscordParticipantScoreUpdated)

			h.Logger.InfoContext(ctx, "Bridged score override success to discord participant score updated",
				attr.RoundID("round_id", p.RoundID),
				attr.String("user_id", string(p.UserID)),
				attr.Int("score", int(p.Score)),
				attr.String("discord_message_id", messageID),
			)
			return []*message.Message{bridgeMsg}, nil
		},
	)(msg)
}
