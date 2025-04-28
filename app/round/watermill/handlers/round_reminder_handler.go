package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
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

			result, err := h.RoundDiscord.GetRoundReminderManager().SendRoundReminder(ctx, reminderPayload)
			if err != nil {
				return nil, fmt.Errorf("failed to send round reminder: %w", err)
			}

			success, ok := result.Success.(bool)
			if !ok {
				success = false
			}

			status := "reminder_failed"
			if success {
				status = "reminder_sent"
			}

			tracePayload := map[string]interface{}{
				"round_id":      reminderPayload.RoundID,
				"reminder_type": reminderPayload.ReminderType,
				"status":        status,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
