package helpers

import (
	"encoding/json"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// PublishEvent publishes an event to the event bus.
func PublishEvent(eventBus message.Publisher, topic string, correlationID string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	return eventBus.Publish(topic, msg)
}

// SendUserMessage sends a message to the user.
func SendUserMessage(s discord.Discord, userID, messageText, correlationID string, getChannelID func(discord.Discord, string) (string, error), errorReporter func(string, string, error)) {
	channelID, err := getChannelID(s, userID)
	if err != nil {
		errorReporter(correlationID, "error getting channel ID", err)
		return
	}
	if _, err = s.ChannelMessageSend(channelID, messageText); err != nil {
		errorReporter(correlationID, "error sending message", err)
	}
}
