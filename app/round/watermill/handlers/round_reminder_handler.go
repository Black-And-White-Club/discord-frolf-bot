package roundhandlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleRoundReminder prepares and sends a round reminder notification.
func (h *RoundHandlers) HandleRoundReminder(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundReminder",
		&roundevents.DiscordReminderPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			reminderPayload, ok := payload.(*roundevents.DiscordReminderPayloadV1)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleRoundReminder")
			}

			// Detect duplicate delivery (JetStream redelivery) and skip if already processed once
			if deliveredStr := msg.Metadata.Get("Delivered"); deliveredStr != "" {
				if deliveredCount, err := strconv.Atoi(deliveredStr); err == nil && deliveredCount > 1 {
					h.Logger.WarnContext(ctx, "Skipping duplicate reminder delivery",
						attr.RoundID("round_id", reminderPayload.RoundID),
						attr.String("message_id", msg.UUID),
						attr.Int("delivered_count", deliveredCount),
					)
					return []*message.Message{}, nil
				}
			}

			// Log basic info for debugging
			h.Logger.InfoContext(ctx, "Processing round reminder",
				attr.RoundID("round_id", reminderPayload.RoundID),
				attr.String("reminder_type", reminderPayload.ReminderType),
				attr.String("message_id", msg.UUID))

			// Early validation - fail fast for invalid payloads
			if reminderPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
				h.Logger.ErrorContext(ctx, "Round ID is required for reminder",
					attr.String("message_id", msg.UUID))
				// Don't return error - this payload is invalid and shouldn't be retried
				return []*message.Message{}, nil
			}

			// Use default channel from config if payload doesn't have one
			if reminderPayload.DiscordChannelID == "" {
				defaultChannelID := h.Config.GetEventChannelID()
				if defaultChannelID == "" {
					h.Logger.ErrorContext(ctx, "No channel ID configured for reminder",
						attr.String("message_id", msg.UUID))
					// Don't return error - configuration issue, no point retrying
					return []*message.Message{}, nil
				}

				h.Logger.InfoContext(ctx, "Using default channel for reminder",
					attr.String("channel_id", defaultChannelID),
					attr.String("message_id", msg.UUID))

				// Create new payload with default channel
				reminderPayload = &roundevents.DiscordReminderPayloadV1{
					RoundID:          reminderPayload.RoundID,
					DiscordChannelID: defaultChannelID,
					ReminderType:     reminderPayload.ReminderType,
					RoundTitle:       reminderPayload.RoundTitle,
					StartTime:        reminderPayload.StartTime,
					Location:         reminderPayload.Location,
					UserIDs:          reminderPayload.UserIDs,
					DiscordGuildID:   reminderPayload.DiscordGuildID,
					EventMessageID:   reminderPayload.EventMessageID,
				}
			}

			// Create timeout context for Discord API call (30 seconds max)
			apiCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			h.Logger.InfoContext(ctx, "Sending Discord reminder",
				attr.RoundID("round_id", reminderPayload.RoundID),
				attr.String("message_id", msg.UUID))

			// CRITICAL OPERATION: Send the Discord reminder with timeout protection
			result, err := h.RoundDiscord.GetRoundReminderManager().SendRoundReminder(apiCtx, reminderPayload)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send Discord reminder",
					attr.Error(err),
					attr.String("message_id", msg.UUID))

				// Check if it's a timeout or context cancellation
				if apiCtx.Err() == context.DeadlineExceeded {
					h.Logger.WarnContext(ctx, "Discord reminder timed out after 30 seconds",
						attr.String("message_id", msg.UUID))
					// Don't retry timeouts - they'll likely timeout again
					return []*message.Message{}, nil
				}

				// For other errors, allow retry by returning error
				return nil, fmt.Errorf("failed to send round reminder: %w", err)
			}

			// Validate result with defensive check
			var success bool
			if successVal, ok := result.Success.(bool); ok {
				success = successVal
			}

			if !success {
				h.Logger.WarnContext(ctx, "Discord reminder operation reported failure",
					attr.String("message_id", msg.UUID))
				// Don't retry operational failures - they indicate the reminder was processed but failed
				return []*message.Message{}, nil
			}

			h.Logger.InfoContext(ctx, "Round reminder sent to Discord successfully",
				attr.RoundID("round_id", reminderPayload.RoundID),
				attr.String("reminder_type", reminderPayload.ReminderType),
				attr.Int("user_count", len(reminderPayload.UserIDs)),
				attr.String("message_id", msg.UUID))

			// Return empty message array to acknowledge successfully
			return []*message.Message{}, nil
		},
	)(msg)
}
