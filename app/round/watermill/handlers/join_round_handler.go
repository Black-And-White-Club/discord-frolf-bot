package roundhandlers

import (
	"context"
	"fmt"

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
		&discordroundevents.DiscordRoundParticipantJoinRequestPayload{}, // Fix: Use the correct Discord payload type
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload) // Fix: Cast to correct type

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
				// Default to accepted if invalid/missing response
				response = roundtypes.ResponseAccept
			}

			// Check if this is a late join (defaults to false if not set)
			joinedLate := false
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			// Set tag number to 0 (backend will assign actual tag numbers)
			tagNumber := sharedtypes.TagNumber(0)

			// Construct the backend payload
			backendPayload := roundevents.ParticipantJoinRequestPayload{
				RoundID:    p.RoundID,
				UserID:     p.UserID,
				Response:   response,
				TagNumber:  &tagNumber,
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
				attr.String("response", responseStr),
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
			// The payload variable is now the unmarshalled *roundevents.ParticipantJoinedPayload
			p := payload.(*roundevents.ParticipantJoinedPayload)

			// --- Add logs here to inspect the unmarshalled payload ---
			h.Logger.InfoContext(ctx, "Received ParticipantJoinedPayload",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", p.RoundID),
				attr.Any("accepted_count_payload", len(p.AcceptedParticipants)),   // Log count from payload
				attr.Any("declined_count_payload", len(p.DeclinedParticipants)),   // Log count from payload
				attr.Any("tentative_count_payload", len(p.TentativeParticipants)), // Log count from payload
				attr.Any("accepted_participants_payload", p.AcceptedParticipants), // Log content of accepted list
				// Add logging for other lists if needed, but accepted is the key one here
			)
			// --- End added logs ---

			channelID := h.Config.Discord.ChannelID // Assuming Config is available
			messageID := msg.Metadata.Get("discord_message_id")

			// Determine if this was a late join
			joinedLate := false
			if p.JoinedLate != nil {
				joinedLate = *p.JoinedLate
			}

			// Update the Discord embed with the new participant information
			// This is where the (potentially empty) slices from the payload are passed
			result, err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed( // Assuming RoundDiscord is available
				ctx,
				channelID,
				messageID,
				p.AcceptedParticipants,
				p.DeclinedParticipants,
				p.TentativeParticipants,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed",
					attr.CorrelationIDFromMsg(msg), // Add correlation ID to error log
					attr.Error(err),
				)
				// You might want to return a message indicating failure back to the user in Discord
				// return nil, err // Return the error so handlerWrapper logs failure
				// Or publish a failure event
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
	// Use the common handler wrapper for tracing, metrics, and error handling
	return h.handlerWrapper(
		"HandleRoundParticipantRemoved",          // Handler name
		&roundevents.ParticipantRemovedPayload{}, // Target payload type for this event
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Unmarshal the specific payload for participant removal
			p := payload.(*roundevents.ParticipantRemovedPayload)

			// Log the received payload, including the counts of the lists
			h.Logger.InfoContext(ctx, "Received RoundParticipantRemoved event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),
				attr.String("user_id", string(p.UserID)),                          // Log the user ID that was removed
				attr.Int("accepted_count_payload", len(p.AcceptedParticipants)),   // Log count from payload
				attr.Int("declined_count_payload", len(p.DeclinedParticipants)),   // Log count from payload
				attr.Int("tentative_count_payload", len(p.TentativeParticipants)), // Log count from payload
				// Optionally log content of lists if needed for debugging
				// attr.Any("accepted_participants_payload", p.AcceptedParticipants),
			)

			// Get channel and message ID (assuming EventMessageID is now in the payload, otherwise get from msg.Metadata)
			// Your UpdateRoundEventEmbed uses channelID and messageID parameters, not from payload/metadata directly
			channelID := h.Config.Discord.ChannelID // Get channel ID from config
			// Assuming discord_message_id is still passed in message metadata for CreateResultMessage
			messageID := msg.Metadata.Get("discord_message_id") // Get message ID from metadata

			// Use the participant lists provided directly in the payload
			acceptedParticipants := p.AcceptedParticipants
			declinedParticipants := p.DeclinedParticipants
			tentativeParticipants := p.TentativeParticipants

			// This calls the SAME function used by HandleRoundParticipantJoined
			_, err := h.RoundDiscord.GetRoundRsvpManager().UpdateRoundEventEmbed(
				ctx,
				channelID,            // Pass channel ID
				messageID,            // Pass message ID
				acceptedParticipants, // Pass the lists from the payload
				declinedParticipants,
				tentativeParticipants,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed after removal",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", p.RoundID.String()),  // Log round ID in error
					attr.String("discord_message_id", messageID), // Log message ID in error
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to update round event embed after removal: %w", err)
			}

			// Log success after the embed is updated
			h.Logger.InfoContext(ctx, "Successfully updated round event embed after removal",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),  // Log round ID on success
				attr.String("discord_message_id", messageID), // Log message ID on success
				// UpdateRoundEventEmbed logs the final counts, so no need to duplicate here unless desired
			)

			// Removal handlers typically don't publish outgoing messages
			return nil, nil
		},
	)(msg) // Execute the wrapped handler
}
