package roundhandlers

// import (
// 	"encoding/json"
// 	"fmt"

// 	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
// 	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// // HandleRoundReminder prepares the reminder and publishes the discord.round.reminder event.
// func (h *RoundHandlers) HandleRoundReminder(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling round reminder", attr.CorrelationIDFromMsg(msg))
// 	var payload roundevents.RoundReminderPayload // This is the *shared* event from the backend.
// 	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
// 		h.Logger.Error(ctx, "Failed to unmarshal RoundReminderPayload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
// 	}
// 	h.Logger.Info(ctx, "Preparing round reminder", attr.Int64("round_id", payload.RoundID), attr.String("reminder_type", payload.ReminderType), attr.CorrelationIDFromMsg(msg))
// 	channelID := msg.Metadata.Get("channel_id")
// 	// Construct the Discord-specific payload
// 	discordPayload := discordroundevents.DiscordRoundReminderPayload{
// 		RoundID:      payload.RoundID,
// 		RoundTitle:   payload.RoundTitle,
// 		UserIDs:      payload.UserIDs,
// 		ReminderType: payload.ReminderType,
// 		ChannelID:    channelID,
// 	}
// 	discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, discordroundevents.RoundReminderTopic) // Use the helper
// 	if err != nil {
// 		return nil, err
// 	}
// 	h.Logger.Info(ctx, "Successfully processed round reminder", attr.CorrelationIDFromMsg(msg))
// 	return []*message.Message{discordMsg}, nil
// }
