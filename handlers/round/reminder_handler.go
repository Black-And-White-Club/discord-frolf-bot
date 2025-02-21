package roundhandlers

import (
	"encoding/json"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SendReminder sends a reminder message to the appropriate Discord channel.
func (h *RoundHandlers) SendReminder(msg *message.Message) {
	var payload roundevents.DiscordReminderPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		h.logger.Error("Failed to unmarshal DiscordReminderPayload", "error", err)
		return
	}

	h.logger.Info("Sending round reminder", "round_id", payload.RoundID, "reminder_type", payload.ReminderType)

	// Create the reminder message
	var reminderMessage string
	switch payload.ReminderType {
	case "1h":
		reminderMessage = "Reminder: The round starts in one hour!"
	default:
		h.logger.Error("Unknown reminder type", "reminder_type", payload.ReminderType)
		return
	}

	// Create a thread in the channel
	threadName := fmt.Sprintf("Round: %s", payload.RoundTitle)
	threadID, err := h.Session.CreateThread(payload.RoundID, threadName)
	if err != nil {
		h.logger.Error("Failed to create thread", "error", err)
		return
	}

	// Add participants to the thread and build the mention string
	var mentions string
	for _, userID := range payload.UserIDs {
		err := h.Session.AddUserToThread(threadID.ID, userID)
		if err != nil {
			h.logger.Error("Failed to add user to thread", "user_id", userID, "error", err)
		} else {
			mentions += fmt.Sprintf("<@%s> ", userID)
		}
	}

	// Send the reminder message in the thread, tagging all participants
	reminderMessage = fmt.Sprintf("%s\n%s", reminderMessage, mentions)
	_, err = h.Session.ChannelMessageSend(threadID.ID, reminderMessage)
	if err != nil {
		h.logger.Error("Failed to send reminder message", "error", err)
	}
}
