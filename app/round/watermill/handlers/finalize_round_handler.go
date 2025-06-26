package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundFinalized (Discord)",                // Descriptive handler name
		&roundevents.RoundFinalizedEmbedUpdatePayload{}, // Expecting this payload from backend
		// Corrected return type here: []message.Message
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Assert the payload to the expected type
			p, ok := payload.(*roundevents.RoundFinalizedEmbedUpdatePayload)
			if !ok {
				h.Logger.ErrorContext(ctx, "Invalid payload type for HandleRoundFinalized (Discord)",
					attr.Any("payload", payload), // Log the received payload
				)
				return nil, fmt.Errorf("invalid payload type for HandleRoundFinalized (Discord)")
			}
			discordChannelID := h.Config.GetEventChannelID()
			// Validate input payload - check for mandatory fields
			if uuid.UUID(p.RoundID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing RoundID in payload for Discord Finalized", attr.CorrelationIDFromMsg(msg)) // Assuming CorrelationIDFromMsg helper
				return nil, fmt.Errorf("missing RoundID in Discord Finalized payload")
			}

			// *** RETRIEVE THE MESSAGE ID FROM THE MESSAGE METADATA ***
			discordMessageID := msg.Metadata.Get("discord_message_id")
			if discordMessageID == "" { // Validate the message ID from metadata
				h.Logger.ErrorContext(ctx, "Missing discord_message_id in message metadata for Discord Finalized",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
				)
				return nil, fmt.Errorf("missing discord_message_id in metadata for round finalized")
			}

			// The ChannelID is needed by FinalizeScorecardEmbed.
			// It should ideally also be in the payload (p.DiscordChannelID), or fetched here if necessary.
			// Let's use the ChannelID from the payload if available, otherwise metadata.
			// Or better, enforce it in the payload for clarity. Assuming it's in payload.
			// if p.DiscordChannelID == "" { // Validate ChannelID from payload
			// 	h.Logger.ErrorContext(ctx, "Missing DiscordChannelID in payload for Discord Finalized",
			// 		attr.CorrelationIDFromMsg(msg),
			// 		attr.RoundID("round_id", p.RoundID),
			// 	)
			// 	return nil, fmt.Errorf("missing DiscordChannelID in round finalized payload")
			// }

			h.Logger.InfoContext(ctx, "Received DiscordRoundFinalized event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", p.RoundID),
				attr.String("discord_message_id_from_metadata", discordMessageID), // Log message ID from metadata
				attr.String("channel_id_from_payload", discordChannelID),          // Log channel ID from payload
			)

			// Log the message ID from the payload as well for comparison/debugging if needed
			if p.EventMessageID != "" && p.EventMessageID != discordMessageID {
				h.Logger.DebugContext(ctx, "Message ID in payload differs from metadata",
					attr.CorrelationIDFromMsg(msg),
					attr.String("message_id_from_payload", p.EventMessageID),
					attr.String("message_id_from_metadata", discordMessageID),
				)
			}

			// --- Trigger Embed Finalization Update ---
			// Get the FinalizeRoundManager on the Discord App
			finalizeRoundManager := h.RoundDiscord.GetFinalizeRoundManager() // Assuming this method exists

			// Finalize the round embed by calling the manager method
			// *** PASS THE MESSAGE ID FROM METADATA AND CHANNEL ID FROM PAYLOAD ***
			finalizeResult, err := finalizeRoundManager.FinalizeScorecardEmbed(
				ctx,
				discordMessageID, // Pass message ID obtained from metadata
				discordChannelID, // Pass channel ID from payload
				*p,               // Pass the full payload struct value (dereferencing p pointer)
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to finalize scorecard embed", attr.Error(err))
				return nil, fmt.Errorf("failed to finalize scorecard embed: %w", err)
			}

			// Check the result from the FinalizeScorecardEmbed operation
			if finalizeResult.Error != nil { // Assuming FinalizeRoundOperationResult uses .Error for operational errors
				h.Logger.ErrorContext(ctx, "FinalizeScorecardEmbed operation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(finalizeResult.Error),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID), // Log message ID used
				)
				return nil, fmt.Errorf("finalize scorecard embed operation failed: %w", finalizeResult.Error)
			}

			h.Logger.InfoContext(ctx, "Successfully finalized round scorecard on Discord",
				attr.RoundID("round_id", p.RoundID),
				attr.String("discord_message_id", discordMessageID), // Log message ID used
				attr.String("channel_id", discordChannelID),         // Log channel ID used
			)

			// Create trace event (optional)
			tracePayload := map[string]interface{}{
				"round_id":           p.RoundID,
				"event_type":         "round_finalized",             // Or Discord specific event type
				"status":             "scorecard_finalized_display", // Clarified status
				"discord_message_id": discordMessageID,              // Use message ID from metadata
				"channel_id":         discordChannelID,              // Use channel ID from payload
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent) // Assuming roundevents.RoundTraceEvent topic
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event for embed finalization", attr.Error(err))
				return []*message.Message{}, nil // Return no messages if trace fails
			}

			return []*message.Message{traceMsg}, nil // Return trace message if successful
		},
	)(msg) // Execute the wrapped handler
}
