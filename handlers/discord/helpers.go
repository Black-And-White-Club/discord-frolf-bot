package discordhandlers

import (
	discordevents "github.com/Black-And-White-Club/discord-frolf-bot/events/discord"
	"github.com/ThreeDotsLabs/watermill/message"
)

// interactionResponded creates a payload for Discord interaction responses.
func (h *DiscordHandlers) interactionResponded(msg *message.Message, userID, status string) discordevents.InteractionRespondedPayload {
	interactionPayload := discordevents.InteractionRespondedPayload{
		InteractionID: msg.Metadata.Get("interaction_id"),
		UserID:        userID,
		Status:        status,
	}

	return interactionPayload
}
