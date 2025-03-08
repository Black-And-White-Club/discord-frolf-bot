package roundhandlers

import (
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round participant join request", attr.CorrelationIDFromMsg(msg))
	var payload discordroundevents.DiscordRoundParticipantJoinRequestPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}
	// Construct the backend payload
	backendPayload := roundevents.ParticipantJoinRequestPayload{
		RoundID:   payload.RoundID,
		DiscordID: payload.UserID,
	}
	backendMsg, err := h.createResultMessage(msg, backendPayload, roundevents.RoundParticipantJoinRequest)
	if err != nil {
		return nil, err
	}
	h.Logger.Info(ctx, "Successfully processed participant join request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

// This comes from the backend
func (h *RoundHandlers) HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling participant joined", attr.CorrelationIDFromMsg(msg))
	var payload roundevents.ParticipantJoinedPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}
	channelID := msg.Metadata.Get("channel_id")
	// Construct the *internal* Discord payload for successful join
	discordPayload := discordroundevents.DiscordRoundParticipantJoinedPayload{
		RoundID:   payload.RoundID,
		UserID:    payload.Participant,
		TagNumber: payload.TagNumber,
		ChannelID: channelID,
	}
	discordMsg, err := h.createResultMessage(msg, discordPayload, discordroundevents.RoundParticipantJoinedTopic)
	if err != nil {
		return nil, err
	}
	h.Logger.Info(ctx, "Successfully processed participant joined", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
