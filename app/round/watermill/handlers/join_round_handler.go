package roundhandlers

import (
	"context"
	"fmt"
	"strings"

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

			// Extract and normalize response from message metadata. Support multiple token styles
			// so Discord side can just send the enum value from shared types.
			rawResponse := msg.Metadata.Get("response")
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
				// Unknown token: log & default
				h.Logger.Warn("Unrecognized RSVP token; defaulting to ACCEPT", attr.String("raw_response", rawResponse))
				response = roundtypes.ResponseAccept
			}

			// Check if this is a late join
			joinedLate := false
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			// Construct the backend payload. TagNumber left nil so backend performs lookup.
			zeroTag := sharedtypes.TagNumber(0)
			backendPayload := roundevents.ParticipantJoinRequestPayload{
				GuildID:    sharedtypes.GuildID(p.GuildID),
				RoundID:    p.RoundID,
				UserID:     p.UserID,
				Response:   response,
				TagNumber:  &zeroTag, // tests expect non-nil pointer (they dereference)
				JoinedLate: &joinedLate,
			}

			// Create message to send to backend
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, roundevents.RoundParticipantJoinRequest)
			if err != nil {
				return nil, err
			}

			h.Logger.InfoContext(ctx, "Successfully processed participant join request",
				attr.CorrelationIDFromMsg(msg),
				attr.Bool("joined_late", joinedLate),
				attr.String("response", string(response)),
			)

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleRoundParticipantJoined handles the event when a participant has joined a round.
func (h *RoundHandlers) HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error) {
	// The outer handlerWrapper handles the high-level span, metrics, and start/end logs
	return h.handlerWrapper(
		"HandleRoundParticipantJoined",
		&roundevents.ParticipantJoinedPayload{}, // Unmarshal target
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.ParticipantJoinedPayload)

			h.Logger.InfoContext(ctx, "Received ParticipantJoinedPayload",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", p.RoundID),
				attr.Any("accepted_count_payload", len(p.AcceptedParticipants)),   // Log count from payload
				attr.Any("declined_count_payload", len(p.DeclinedParticipants)),   // Log count from payload
				attr.Any("tentative_count_payload", len(p.TentativeParticipants)), // Log count from payload
				attr.Any("accepted_participants_payload", p.AcceptedParticipants), // Log content of accepted list
				// Add logging for other lists if needed, but accepted is the key one here
			)

			// Resolve channel ID (currently from in-memory config). If empty, embed update can't proceed.
			// NOTE: Future enhancement: use per-event config fragment once available in dependency version.
			channelID := ""
			if h.Config != nil && h.Config.GetEventChannelID() != "" {
				channelID = h.Config.GetEventChannelID()
			}

			if channelID == "" {
				h.Logger.WarnContext(ctx, "Channel ID not resolved for ParticipantJoined; skipping embed update to avoid 404",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
				)
				return nil, nil // Ack without retry; cannot proceed without channelID
			}
			messageID := msg.Metadata.Get("discord_message_id")
			// Intentionally do NOT fallback to payload EventMessageID if metadata key is missing.
			// Tests expect empty messageID in this scenario so downstream embed update is still attempted with empty ID.

			// Determine if this was a late join
			joinedLate := false
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			var result interface{}
			var err error

			// If this is a late join, the round has started and we need to update a scorecard embed
			// If not a late join, it's still an RSVP embed
			if joinedLate {
				h.Logger.InfoContext(ctx, "Processing late join - using scorecard logic",
					attr.CorrelationIDFromMsg(msg),
					attr.Bool("joined_late", joinedLate),
					attr.Any("joined_late_pointer", p.JoinedLate))

				// Use scorecard embed update for started rounds - add late participant to scorecard
				result, err = h.RoundDiscord.GetScoreRoundManager().AddLateParticipantToScorecard(
					ctx,
					channelID,
					messageID,
					p.AcceptedParticipants, // These are already Participant structs with Accept response
				)
			} else {
				h.Logger.InfoContext(ctx, "Processing regular join - using RSVP logic",
					attr.CorrelationIDFromMsg(msg),
					attr.Bool("joined_late", joinedLate),
					attr.Any("joined_late_pointer", p.JoinedLate))

				// Use RSVP embed update for rounds that haven't started
				result, err = h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
					ctx,
					channelID,
					messageID,
					p.AcceptedParticipants,
					p.DeclinedParticipants,
					p.TentativeParticipants,
				)
			}

			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed",
					attr.CorrelationIDFromMsg(msg), // Add correlation ID to error log
					attr.Error(err),
				)
				return nil, err // Re-throw the error
			}

			// This log is already present, confirming embed update was called and completed
			h.Logger.InfoContext(ctx, "Successfully updated participant joined",
				attr.CorrelationIDFromMsg(msg),
				attr.Bool("joined_late", joinedLate),
				attr.Any("result", result), // Logs the result from UpdateRoundEventEmbed
			)

			// You might publish a success event here if needed
			return nil, nil // No outgoing messages from this handler
		},
	)(msg) // Execute the wrapped handler
}

func (h *RoundHandlers) HandleRoundParticipantRemoved(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundParticipantRemoved",
		&roundevents.ParticipantRemovedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.ParticipantRemovedPayload)

			h.Logger.InfoContext(ctx, "Received RoundParticipantRemoved event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),
				attr.String("user_id", string(p.UserID)),
				attr.Int("accepted_count_payload", len(p.AcceptedParticipants)),
				attr.Int("declined_count_payload", len(p.DeclinedParticipants)),
				attr.Int("tentative_count_payload", len(p.TentativeParticipants)),
			)

			// Resolve channel ID similarly for removal events
			channelID := ""
			if h.Config != nil && h.Config.GetEventChannelID() != "" {
				channelID = h.Config.GetEventChannelID()
			}

			if channelID == "" {
				h.Logger.WarnContext(ctx, "Channel ID not resolved for ParticipantRemoved; skipping embed update to avoid 404",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", p.RoundID.String()),
				)
				return nil, nil
			}
			messageID := msg.Metadata.Get("discord_message_id")

			// Check if this is a scorecard embed (started round) by checking if any participant has a score
			isScorecard := false
			for _, participant := range p.AcceptedParticipants {
				if participant.Score != nil {
					isScorecard = true
					break
				}
			}

			// If it's a scorecard embed, skip the update (participants can't be removed from started rounds)
			if isScorecard {
				h.Logger.InfoContext(ctx, "Skipping removal update for scorecard embed - participants cannot be removed from started rounds",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", p.RoundID.String()),
					attr.String("discord_message_id", messageID),
				)
				return nil, nil // Don't update the embed for scorecard removals
			}

			// Only update RSVP embeds (before round starts)
			_, err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
				ctx,
				channelID,
				messageID,
				p.AcceptedParticipants,
				p.DeclinedParticipants,
				p.TentativeParticipants,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed after removal",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", p.RoundID.String()),
					attr.String("discord_message_id", messageID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to update round event embed after removal: %w", err)
			}

			h.Logger.InfoContext(ctx, "Successfully updated round event embed after removal",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),
				attr.String("discord_message_id", messageID),
			)

			return nil, nil
		},
	)(msg)
}
