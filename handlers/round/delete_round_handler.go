package roundhandlers

import (
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round delete request", attr.CorrelationIDFromMsg(msg))

	var payload discordroundevents.DiscordRoundDeleteRequestPayload // Need to create this
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err // unmarshalPayload already logs
	}

	// Construct the backend payload
	backendPayload := roundevents.RoundDeleteRequestPayload{
		RoundID:                 payload.RoundID,
		RequestingUserDiscordID: payload.UserID, // Include the requesting user
	}

	backendMsg, err := h.createResultMessage(msg, backendPayload, roundevents.RoundDeleteRequest)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round delete request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

func (h *RoundHandlers) HandleRoundDeleted(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round deleted event", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.RoundDeletedPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Construct the *internal* Discord payload
	discordPayload := discordroundevents.DiscordRoundDeletedPayload{
		RoundID: payload.RoundID, // Comes from the backend payload
	}

	discordMsg, err := h.createResultMessage(msg, discordPayload, discordroundevents.RoundDeletedTopic)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round deleted", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
