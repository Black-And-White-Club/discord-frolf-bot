package roundhandlers

import (
	"context"
	"fmt"

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
		&roundevents.DiscordReminderPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			reminderPayload, ok := payload.(*roundevents.DiscordReminderPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for HandleRoundReminder")
			}

			// Terminal debug logs
			fmt.Printf("\n=== HANDLER START ===\n")
			fmt.Printf("Message ID: %s\n", msg.UUID)
			fmt.Printf("Round ID: %s\n", reminderPayload.RoundID)
			fmt.Printf("Reminder Type: %s\n", reminderPayload.ReminderType)
			fmt.Printf("Delivered: %s\n", msg.Metadata.Get("Delivered"))
			fmt.Printf("ConsumerSeq: %s\n", msg.Metadata.Get("ConsumerSeq"))
			fmt.Printf("StreamSeq: %s\n", msg.Metadata.Get("StreamSeq"))
			fmt.Printf("Consumer: %s\n", msg.Metadata.Get("Consumer"))
			fmt.Printf("Stream: %s\n", msg.Metadata.Get("Stream"))
			fmt.Printf("Timestamp: %s\n", msg.Metadata.Get("Timestamp"))
			fmt.Printf("All Metadata: %+v\n", msg.Metadata)
			fmt.Printf("======================\n")

			// Early validation
			if reminderPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
				fmt.Printf("❌ VALIDATION FAILED: Round ID is required\n")
				return nil, fmt.Errorf("round ID is required")
			}

			// Use default channel from config if payload doesn't have one
			if reminderPayload.DiscordChannelID == "" {
				defaultChannelID := h.Config.Discord.EventChannelID
				if defaultChannelID == "" {
					fmt.Printf("❌ VALIDATION FAILED: No channel ID configured\n")
					return nil, fmt.Errorf("no channel_id in payload and no default event_channel_id in config")
				}

				fmt.Printf("ℹ️  Using default channel: %s\n", defaultChannelID)

				// Create new payload with default channel
				reminderPayload = &roundevents.DiscordReminderPayload{
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

			fmt.Printf("🚀 SENDING DISCORD REMINDER...\n")

			// CRITICAL OPERATION: Send the Discord reminder
			result, err := h.RoundDiscord.GetRoundReminderManager().SendRoundReminder(ctx, reminderPayload)
			if err != nil {
				fmt.Printf("❌ DISCORD SEND FAILED: %v\n", err)
				return nil, fmt.Errorf("failed to send round reminder: %w", err)
			}

			// Validate result
			success, ok := result.Success.(bool)
			if !ok {
				success = false
			}

			if !success {
				fmt.Printf("❌ DISCORD OPERATION REPORTED FAILURE\n")
				return nil, fmt.Errorf("discord reminder operation reported failure")
			}

			fmt.Printf("✅ DISCORD REMINDER SENT SUCCESSFULLY\n")

			// Log successful completion
			h.Logger.InfoContext(ctx, "Round reminder sent to Discord successfully",
				attr.RoundID("round_id", reminderPayload.RoundID),
				attr.String("reminder_type", reminderPayload.ReminderType),
				attr.Int("user_count", len(reminderPayload.UserIDs)),
				attr.String("message_id", msg.UUID))

			fmt.Printf("=== HANDLER END SUCCESS ===\n")
			fmt.Printf("Message ID: %s\n", msg.UUID)
			fmt.Printf("Round ID: %s\n", reminderPayload.RoundID)
			fmt.Printf("Returning: []*message.Message{} (empty slice)\n")
			fmt.Printf("============================\n\n")

			// Return empty message array to acknowledge immediately
			return []*message.Message{}, nil
		},
	)(msg)
}
