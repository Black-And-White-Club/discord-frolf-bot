package handlers

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundParticipantJoinRequest handles the request for a participant to join a round.
func (h *RoundHandlers) HandleRoundParticipantJoinRequest(ctx context.Context, payload *discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) ([]handlerwrapper.Result, error) {
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
	// Resolve channel ID from guild config
	var channelID string
	if h.guildConfigResolver != nil {
		guildCfg, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
		if err != nil || guildCfg == nil {
			h.logger.WarnContext(ctx, "failed to resolve guild config for participant join, falling back to global config",
				attr.String("guild_id", string(payload.GuildID)),
				attr.Error(err))
			if h.config != nil {
				channelID = h.config.GetEventChannelID()
			}
		} else {
			channelID = guildCfg.EventChannelID
		}
	} else if h.config != nil {
		channelID = h.config.GetEventChannelID()
	}

	if channelID == "" {
		return nil, nil // Ack without retry; cannot proceed without channelID
	}

	// Get message ID from context (set by wrapper from message metadata)
	messageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || messageID == "" {
		if id, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			messageID = id
		} else {
			messageID = ""
		}
	}

	// Determine if this was a late join
	joinedLate := false
	if payload.JoinedLate != nil {
		joinedLate = *payload.JoinedLate
	}

	var updateErr error

	// If this is a late join, the round has started and we need to update a scorecard embed
	// If not a late join, it's still an RSVP embed
	if joinedLate {
		// Use scorecard embed update for started rounds - add late participant to scorecard
		_, updateErr = h.service.GetScoreRoundManager().AddLateParticipantToScorecard(
			ctx,
			channelID,
			messageID,
			payload.AcceptedParticipants, // These are already Participant structs with Accept response
		)
	} else {
		// Use RSVP embed update for rounds that haven't started
		// Merge all participant lists into one for the single Participants field
		allParticipants := append(append(payload.AcceptedParticipants, payload.DeclinedParticipants...), payload.TentativeParticipants...)
		_, updateErr = h.service.GetRoundRsvpManager().UpdateRoundEventEmbed(
			ctx,
			channelID,
			messageID,
			allParticipants,
		)
	}

	if updateErr != nil {
		return nil, fmt.Errorf("failed to update round event embed: %w", updateErr)
	}

	return nil, nil
}

func (h *RoundHandlers) HandleRoundParticipantRemoved(ctx context.Context, payload *roundevents.ParticipantRemovedPayloadV1) ([]handlerwrapper.Result, error) {
	// Resolve channel ID similarly for removal events
	var removeChannelID string
	if h.guildConfigResolver != nil {
		removeGuildCfg, removeErr := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
		if removeErr != nil || removeGuildCfg == nil {
			h.logger.WarnContext(ctx, "failed to resolve guild config for participant removal, falling back to global config",
				attr.String("guild_id", string(payload.GuildID)),
				attr.Error(removeErr))
			if h.config != nil {
				removeChannelID = h.config.GetEventChannelID()
			}
		} else {
			removeChannelID = removeGuildCfg.EventChannelID
		}
	} else if h.config != nil {
		removeChannelID = h.config.GetEventChannelID()
	}

	if removeChannelID == "" {
		return nil, nil
	}

	// Get message ID from context (set by wrapper from message metadata)
	messageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || messageID == "" {
		if id, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			messageID = id
		} else {
			messageID = ""
		}
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
	// Merge all participant lists into one for the single Participants field
	allParticipants := append(append(payload.AcceptedParticipants, payload.DeclinedParticipants...), payload.TentativeParticipants...)
	_, embedErr := h.service.GetRoundRsvpManager().UpdateRoundEventEmbed(
		ctx,
		removeChannelID,
		messageID,
		allParticipants,
	)
	if embedErr != nil {
		return nil, fmt.Errorf("failed to update round event embed after removal: %w", embedErr)
	}

	return nil, nil
}
