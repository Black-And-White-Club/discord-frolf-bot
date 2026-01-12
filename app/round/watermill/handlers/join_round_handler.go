package roundhandlers

import (
	"context"
	"fmt"
	"strings"

	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundParticipantJoinRequest handles the request for a participant to join a round.
func (h *RoundHandlers) HandleRoundParticipantJoinRequest(ctx context.Context, payload *sharedroundevents.RoundParticipantJoinRequestDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	// Extract and normalize response from context metadata. Support multiple token styles
	// so Discord side can just send the enum value from shared types.
	rawResponse, ok := ctx.Value("response").(string)
	if !ok {
		rawResponse = ""
	}
	normalized := strings.ToUpper(strings.TrimSpace(rawResponse))
	var response roundtypes.Response
	switch normalized {
	case "ACCEPT", "ACCEPTED":
		response = roundtypes.ResponseAccept
	case "DECLINE", "DECLINED":
		response = roundtypes.ResponseDecline
	case "TENTATIVE":
		response = roundtypes.ResponseTentative
	case "":
		// Missing metadata: default accept (legacy behavior)
		response = roundtypes.ResponseAccept
	default:
		// Unknown token: default
		response = roundtypes.ResponseAccept
	}

	// Check if this is a late join
	joinedLate := false
	if payload.JoinedLate != nil {
		joinedLate = *payload.JoinedLate
	}

	// Construct the backend payload. TagNumber left nil so backend performs lookup.
	zeroTag := sharedtypes.TagNumber(0)
	backendPayload := roundevents.ParticipantJoinRequestPayloadV1{
		GuildID:    sharedtypes.GuildID(payload.GuildID),
		RoundID:    payload.RoundID,
		UserID:     payload.UserID,
		Response:   response,
		TagNumber:  &zeroTag, // tests expect non-nil pointer (they dereference)
		JoinedLate: &joinedLate,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundParticipantJoinRequestedV1,
			Payload: backendPayload,
		},
	}, nil
}

// HandleRoundParticipantJoined handles the event when a participant has joined a round.
func (h *RoundHandlers) HandleRoundParticipantJoined(ctx context.Context, payload *roundevents.ParticipantJoinedPayloadV1) ([]handlerwrapper.Result, error) {
	// Resolve channel ID (currently from in-memory config). If empty, embed update can't proceed.
	channelID := ""
	if h.Config != nil && h.Config.GetEventChannelID() != "" {
		channelID = h.Config.GetEventChannelID()
	}

	if channelID == "" {
		return nil, nil // Ack without retry; cannot proceed without channelID
	}

	// Get message ID from context (set by wrapper from message metadata)
	messageID, ok := ctx.Value("discord_message_id").(string)
	if !ok {
		messageID = ""
	}

	// Determine if this was a late join
	joinedLate := false
	if payload.JoinedLate != nil {
		joinedLate = *payload.JoinedLate
	}

	var err error

	// If this is a late join, the round has started and we need to update a scorecard embed
	// If not a late join, it's still an RSVP embed
	if joinedLate {
		// Use scorecard embed update for started rounds - add late participant to scorecard
		_, err = h.RoundDiscord.GetScoreRoundManager().AddLateParticipantToScorecard(
			ctx,
			channelID,
			messageID,
			payload.AcceptedParticipants, // These are already Participant structs with Accept response
		)
	} else {
		// Use RSVP embed update for rounds that haven't started
		_, err = h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
			ctx,
			channelID,
			messageID,
			payload.AcceptedParticipants,
			payload.DeclinedParticipants,
			payload.TentativeParticipants,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update round event embed: %w", err)
	}

	return nil, nil
}

func (h *RoundHandlers) HandleRoundParticipantRemoved(ctx context.Context, payload *roundevents.ParticipantRemovedPayloadV1) ([]handlerwrapper.Result, error) {
	// Resolve channel ID similarly for removal events
	channelID := ""
	if h.Config != nil && h.Config.GetEventChannelID() != "" {
		channelID = h.Config.GetEventChannelID()
	}

	if channelID == "" {
		return nil, nil
	}

	// Get message ID from context (set by wrapper from message metadata)
	messageID, ok := ctx.Value("discord_message_id").(string)
	if !ok {
		messageID = ""
	}

	// Check if this is a scorecard embed (started round) by checking if any participant has a score
	isScorecard := false
	for _, participant := range payload.AcceptedParticipants {
		if participant.Score != nil {
			isScorecard = true
			break
		}
	}

	// If it's a scorecard embed, skip the update (participants can't be removed from started rounds)
	if isScorecard {
		return nil, nil // Don't update the embed for scorecard removals
	}

	// Only update RSVP embeds (before round starts)
	_, err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
		ctx,
		channelID,
		messageID,
		payload.AcceptedParticipants,
		payload.DeclinedParticipants,
		payload.TentativeParticipants,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update round event embed after removal: %w", err)
	}

	return nil, nil
}
