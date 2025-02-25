package userhandlers

import (
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	userDomain                = "discord_user"
	interactionRespondedTopic = "discord.interaction.responded"
)

// interactionResponded creates a payload for Discord interaction responses.
func (h *UserHandlers) interactionResponded(msg *message.Message, userID, status string) discorduserevents.InteractionRespondedPayload {
	interactionPayload := discorduserevents.InteractionRespondedPayload{
		InteractionID: msg.Metadata.Get("interaction_id"),
		UserID:        userID,
		Status:        status,
	}

	return interactionPayload
}
