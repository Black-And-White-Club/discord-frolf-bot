package roundhandlers

import (
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundScoreUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round score update request", attr.CorrelationIDFromMsg(msg))

	var payload discordroundevents.DiscordRoundScoreUpdateRequestPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err // unmarshalPayload already logs
	}

	// Construct the backend payload
	backendPayload := roundevents.ScoreUpdateRequestPayload{
		RoundID:     payload.RoundID,
		Participant: payload.UserID,
		Score:       &payload.Score,
	}

	backendMsg, err := h.createResultMessage(msg, backendPayload, roundevents.RoundScoreUpdateRequest)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round score update request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

func (h *RoundHandlers) HandleRoundParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round participant score updated", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.ParticipantScoreUpdatedPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}
	// Construct the *internal* Discord payload
	discordPayload := discordroundevents.DiscordRoundParticipantScoreUpdatedPayload{
		RoundID:   payload.RoundID,
		UserID:    payload.Participant,
		Score:     payload.Score,
		ChannelID: payload.ChannelID,
		MessageID: payload.MessageID,
	}

	discordMsg, err := h.createResultMessage(msg, discordPayload, discordroundevents.RoundParticipantScoreUpdatedTopic)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round participant score updated", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
