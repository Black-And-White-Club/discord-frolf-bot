package roundhandlers

import (
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round update request", attr.CorrelationIDFromMsg(msg))

	var payload discordroundevents.DiscordRoundUpdateRequestPayload
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Construct the backend payload.
	backendPayload := roundevents.RoundUpdateRequestPayload{
		RoundID:        payload.RoundID,
		DiscordEventID: payload.MessageID,
	}
	if payload.Title != nil {
		backendPayload.Title = payload.Title
	}
	if payload.Description != nil {
		backendPayload.Description = payload.Description
	}
	if payload.StartTime != nil {
		backendPayload.StartTime = payload.StartTime
	}

	if payload.Location != nil {
		backendPayload.Location = payload.Location
	}

	backendMsg, err := h.createResultMessage(msg, backendPayload, roundevents.RoundUpdateRequest)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round update request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

func (h *RoundHandlers) HandleRoundUpdated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round updated event", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.RoundUpdatedPayload // From the BACKEND
	if err := h.unmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Construct the *internal* Discord payload for the successful update.
	discordPayload := discordroundevents.DiscordRoundUpdatedPayload{
		RoundID:     payload.RoundID,
		MessageID:   payload.DiscordEventID,
		ChannelID:   payload.ChannelID,
		Title:       payload.Title,
		Description: payload.Description,
		StartTime:   payload.StartTime,
		Location:    payload.Location,
	}

	discordMsg, err := h.createResultMessage(msg, discordPayload, discordroundevents.RoundUpdatedTopic)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed round updated", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
