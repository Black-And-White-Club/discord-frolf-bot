package roundrsvp

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/google/uuid"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundResponse handles the interaction with the Discord API.
func (rrm *roundRsvpManager) HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_round_response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	rrm.logger.InfoContext(ctx, "Handling round RSVP",
		attr.UserID(sharedtypes.DiscordID(userID)))

	// Add nil check for Message before accessing ID
	var messageID string
	if i.Message != nil {
		messageID = i.Message.ID
	}

	rrm.logger.InfoContext(ctx, "Processing RSVP interaction",
		attr.String("interaction_id", i.ID),
		attr.String("discord_message_id", messageID),
	)

	return rrm.operationWrapper(ctx, "handle_round_response", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		// Get user from either Member or direct User field
		var user *discordgo.User
		if i.Member != nil && i.Member.User != nil {
			user = i.Member.User
		} else if i.User != nil {
			user = i.User
		} else {
			return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
		}

		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			return RoundRsvpOperationResult{Error: fmt.Errorf("invalid custom ID: %s", customID)}, nil
		}
		eventIDStr := parts[1]

		var response roundtypes.Response
		switch {
		case strings.HasPrefix(customID, "round_accept"):
			response = roundtypes.ResponseAccept
		case strings.HasPrefix(customID, "round_decline"):
			response = roundtypes.ResponseDecline
		case strings.HasPrefix(customID, "round_tentative"):
			response = roundtypes.ResponseTentative
		default:
			return RoundRsvpOperationResult{Error: fmt.Errorf("unknown response type: %s", customID)}, nil
		}

		if err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		}); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		roundUUID, err := uuid.Parse(eventIDStr)
		if err != nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("failed to parse round UUID: %w", err)}, nil
		}

		var tagNumberPtr *sharedtypes.TagNumber = nil

		payload := roundevents.ParticipantJoinRequestPayload{
			RoundID:   sharedtypes.RoundID(roundUUID),
			UserID:    sharedtypes.DiscordID(user.ID),
			Response:  response,
			TagNumber: tagNumberPtr,
		}

		msg := &message.Message{
			Metadata: message.Metadata{
				"discord_message_id": messageID, // Use the safely extracted messageID
				"topic":              discordroundevents.RoundParticipantJoinReqTopic,
			},
		}

		resultMsg, err := rrm.helper.CreateResultMessage(msg, payload, discordroundevents.RoundParticipantJoinReqTopic)
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		if err := rrm.publisher.Publish(discordroundevents.RoundParticipantJoinReqTopic, resultMsg); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		_, err = rrm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("You have chosen: %s", string(response)),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		return RoundRsvpOperationResult{Success: "RSVP successfully processed"}, nil
	})
}

// InteractionJoinRoundLate handles the interaction when a user clicks "Join Round LATE"
func (rrm *roundRsvpManager) InteractionJoinRoundLate(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "join_round_late")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	rrm.logger.InfoContext(ctx, "Handling late round join", attr.UserID(sharedtypes.DiscordID(userID)))

	return rrm.operationWrapper(ctx, "join_round_late", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		// Get user from either Member or direct User field
		var user *discordgo.User
		if i.Member != nil && i.Member.User != nil {
			user = i.Member.User
		} else if i.User != nil {
			user = i.User
		} else {
			return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
		}

		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			return RoundRsvpOperationResult{Error: fmt.Errorf("invalid custom ID format: %s", customID)}, nil
		}

		roundUUIDStr := strings.TrimPrefix(parts[1], "round-")
		roundUUID, err := uuid.Parse(roundUUIDStr)
		if err != nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("failed to parse round UUID: %w", err)}, nil
		}

		if err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		}); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		tagNumber := sharedtypes.TagNumber(0)
		joinedLate := true

		payload := roundevents.ParticipantJoinRequestPayload{
			RoundID:    sharedtypes.RoundID(roundUUID),
			UserID:     sharedtypes.DiscordID(user.ID),
			Response:   roundtypes.ResponseAccept,
			TagNumber:  &tagNumber,
			JoinedLate: &joinedLate,
		}

		// Add nil check for Message before accessing ID
		var messageID string
		if i.Message != nil {
			messageID = i.Message.ID
		}

		msg := &message.Message{
			Metadata: message.Metadata{
				"discord_message_id": messageID, // Use the safely extracted messageID
				"topic":              discordroundevents.RoundParticipantJoinReqTopic,
			},
		}

		resultMsg, err := rrm.helper.CreateResultMessage(msg, payload, discordroundevents.RoundParticipantJoinReqTopic)
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		if err := rrm.publisher.Publish(discordroundevents.RoundParticipantJoinReqTopic, resultMsg); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		_, err = rrm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You have joined the round!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		return RoundRsvpOperationResult{Success: "Late join successfully processed"}, nil
	})
}
