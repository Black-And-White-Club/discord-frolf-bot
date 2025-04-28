package roundhandlers

import (
	"context"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundParticipantJoinRequest handles the request for a participant to join a round.
func (h *RoundHandlers) HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundParticipantJoinRequest",
		&discordroundevents.DiscordRoundParticipantJoinRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload)

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
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			// Default tag number to 0
			tagNumber := sharedtypes.TagNumber(0)
			tagNumberPtr := &tagNumber

			// Construct the backend payload
			backendPayload := roundevents.ParticipantJoinRequestPayload{
				RoundID:    sharedtypes.RoundID(p.RoundID),
				UserID:     sharedtypes.DiscordID(p.UserID),
				Response:   response,
				TagNumber:  tagNumberPtr,
				JoinedLate: &joinedLate,
			}

			// Create a message to send to the backend
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, roundevents.RoundParticipantJoinRequest)
			if err != nil {
				return nil, err
			}

			h.Logger.InfoContext(ctx, "Successfully processed participant join request",
				attr.CorrelationIDFromMsg(msg),
				attr.Bool("joined_late", joinedLate),
			)

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleRoundParticipantJoined handles the event when a participant has joined a round.
func (h *RoundHandlers) HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundParticipantJoined",
		&roundevents.ParticipantJoinedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.ParticipantJoinedPayload)

			channelID := h.Config.Discord.ChannelID
			messageID := p.EventMessageID

			// Determine if this was a late join
			joinedLate := false
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			// Update the Discord embed with the new participant information
			result, err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
				ctx,
				channelID,
				messageID,
				p.AcceptedParticipants,
				p.DeclinedParticipants,
				p.TentativeParticipants,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed", attr.Error(err))
				return nil, err
			}

			h.Logger.InfoContext(ctx, "Successfully updated participant joined",
				attr.CorrelationIDFromMsg(msg),
				attr.Bool("joined_late", joinedLate),
				attr.Any("result", result))

			return nil, nil
		},
	)(msg)
}
