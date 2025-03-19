package roundhandlers

import (
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundParticipantJoinRequest handles the request for a participant to join a round.
func (h *RoundHandlers) HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round participant join request", attr.CorrelationIDFromMsg(msg))

	var payload discordroundevents.DiscordRoundParticipantJoinRequestPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Extract the response from the message metadata
	responseStr := msg.Metadata.Get("response")
	var response roundtypes.Response
	switch responseStr {
	case "accepted":
		response = roundtypes.ResponseAccept
	case "declined":
		response = roundtypes.ResponseDecline
	case "tentative":
		response = roundtypes.ResponseTentative
	default:
		response = roundtypes.ResponseAccept
	}

	// Check if this is a late join (defaults to false if not set)
	joinedLate := false
	if payload.JoinedLate != nil {
		joinedLate = *payload.JoinedLate
	}

	// Default tag number to 0
	tagNumber := 0
	tagNumberPtr := &tagNumber

	// Construct the backend payload
	backendPayload := roundevents.ParticipantJoinRequestPayload{
		RoundID:    roundtypes.ID(payload.RoundID),
		UserID:     roundtypes.UserID(payload.UserID),
		Response:   response,
		TagNumber:  tagNumberPtr,
		JoinedLate: &joinedLate,
	}

	// Create a message to send to the backend
	backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, roundevents.RoundParticipantJoinRequest)
	if err != nil {
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully processed participant join request",
		attr.CorrelationIDFromMsg(msg),
		attr.Bool("joined_late", joinedLate),
	)

	return []*message.Message{backendMsg}, nil
}

// HandleRoundParticipantJoined handles the event when a participant has joined a round.
func (h *RoundHandlers) HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling participant joined", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.ParticipantJoinedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	channelID := msg.Metadata.Get("channel_id")
	messageID := msg.Metadata.Get("message_id")

	// Determine if this was a late join
	joinedLate := false
	if payload.JoinedLate != nil {
		joinedLate = *payload.JoinedLate
	}

	// Update the Discord embed with the new participant information
	err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
		channelID, messageID,
		payload.AcceptedParticipants, payload.DeclinedParticipants, payload.TentativeParticipants,
	)
	if err != nil {
		h.Logger.Error(ctx, "Failed to update round event embed", attr.Error(err))
		return nil, err
	}

	h.Logger.Info(ctx, "Successfully updated participant joined",
		attr.CorrelationIDFromMsg(msg),
		attr.Bool("joined_late", joinedLate),
	)

	return nil, nil
}
