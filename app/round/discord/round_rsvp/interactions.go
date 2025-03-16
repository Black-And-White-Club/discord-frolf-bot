package roundrsvp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundResponse handles the interaction with the Discord API.
func (rrm *roundRsvpManager) HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate) {
	user := i.Member.User
	eventIDStr := strings.Split(i.MessageComponentData().CustomID, "|")[1]
	var response roundtypes.Response

	switch {
	case strings.HasPrefix(i.MessageComponentData().CustomID, "round_accept"):
		response = roundtypes.ResponseAccept
	case strings.HasPrefix(i.MessageComponentData().CustomID, "round_decline"):
		response = roundtypes.ResponseDecline
	case strings.HasPrefix(i.MessageComponentData().CustomID, "round_tentative"):
		response = roundtypes.ResponseTentative
	default:
		slog.Error("Unknown response type", attr.String("custom_id", i.MessageComponentData().CustomID))
		return
	}

	slog.Info("Processing round RSVP",
		attr.String("user", user.Username),
		attr.String("response", string(response)),
		attr.String("event_id", eventIDStr),
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID))

	// Acknowledge the interaction (so Discord doesn't time out)
	err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		slog.Error("Failed to acknowledge interaction", attr.Error(err))
		return
	}

	// Convert string eventID to int64
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil {
		slog.Error("Failed to parse event ID", attr.Error(err), attr.String("event_id_str", eventIDStr))
		return
	}

	// Convert int64 to roundtypes.ID
	roundID := roundtypes.ID(eventID)
	userID := roundtypes.UserID(user.ID)

	// Create a pointer to an int for TagNumber
	tagNumber := 0
	tagNumberPtr := &tagNumber

	// Create the payload for the backend
	payload := roundevents.ParticipantJoinRequestPayload{
		RoundID:   roundID,
		UserID:    userID,
		Response:  response, // Use the roundtypes.Response type directly
		TagNumber: tagNumberPtr,
	}

	// Create a message to send to the backend
	msg := &message.Message{
		Metadata: message.Metadata{
			"correlation_id": i.ID,
			"topic":          roundevents.RoundParticipantJoinRequest, // Add topic to metadata
		},
	}

	// Call CreateResultMessage with the correct parameters
	resultMsg, err := rrm.helper.CreateResultMessage(msg, payload, roundevents.RoundParticipantJoinRequest)
	if err != nil {
		slog.Error("Failed to create result message", attr.Error(err))
		return
	}

	// Publish the message to the backend
	if err := rrm.publisher.Publish(roundevents.RoundParticipantJoinRequest, resultMsg); err != nil {
		slog.Error("Failed to publish participant join request", attr.Error(err))
		return
	}

	slog.Info("Successfully published participant join request",
		attr.String("message_id", resultMsg.UUID),
		attr.String("topic", roundevents.RoundParticipantJoinRequest))

	// Send a follow-up message to the user (ephemeral)
	_, err = rrm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("You have chosen: %s", string(response)),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		slog.Error("Failed to send follow-up message", attr.Error(err))
	} else {
		slog.Info("Sent ephemeral follow-up message to user", attr.String("user", user.Username))
	}
}
