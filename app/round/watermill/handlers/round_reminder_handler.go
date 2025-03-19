package roundhandlers

import (
	"encoding/json"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundReminder prepares the reminder and publishes the discord.round.reminder event.
func (h *RoundHandlers) HandleRoundReminder(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round reminder", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.DiscordReminderPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal DiscordReminderPayload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	h.Logger.Info(ctx, "Preparing round reminder",
		attr.Int64("round_id", int64(payload.RoundID)),
		attr.String("reminder_type", payload.ReminderType),
		attr.CorrelationIDFromMsg(msg))

	success, err := h.RoundDiscord.GetRoundReminderManager().SendRoundReminder(ctx, &payload)
	if err != nil {
		h.Logger.Error(ctx, "Failed to send round reminder", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to send round reminder: %w", err)
	}

	var traceMsg *message.Message
	if success {
		h.Logger.Info(ctx, "Successfully processed round reminder", attr.CorrelationIDFromMsg(msg))
		traceMsg, err = h.createTraceEventMessage(msg, int64(payload.RoundID), payload.ReminderType, "reminder_sent")
		if err != nil {
			return nil, err
		}
	} else {
		h.Logger.Warn(ctx, "Reminder processing failed (but no error returned)", attr.CorrelationIDFromMsg(msg))
		traceMsg, err = h.createTraceEventMessage(msg, int64(payload.RoundID), payload.ReminderType, "reminder_failed")
		if err != nil {
			return nil, err
		}
	}

	return []*message.Message{traceMsg}, nil
}

func (h *RoundHandlers) createTraceEventMessage(originalMsg *message.Message, roundID int64, reminderType string, status string) (*message.Message, error) {
	tracePayload := map[string]interface{}{
		"round_id":      roundID,
		"reminder_type": reminderType,
		"status":        status,
	}

	return h.Helpers.CreateResultMessage(originalMsg, tracePayload, roundevents.RoundTraceEvent)
}
