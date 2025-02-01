package userhandlers

import (
	"encoding/json"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleReaction handles the reaction event that starts the signup process.
func (h *UserHandlers) HandleReaction(s discord.Discord, r *discord.MessageReactionAdd) {
	// Get the bot user
	botUser, err := s.GetBotUser()
	if err != nil {
		h.ErrorReporter.ReportError("", "error getting bot user", err)
		return
	}

	if r.UserID == botUser.ID {
		return
	}

	// Get the channel ID (after bot user check)
	userChannelID, err := h.getChannelID(s, r.UserID)
	if err != nil {
		h.ErrorReporter.ReportError("", "error getting or creating channel", err, "user_id", r.UserID)
		return
	}

	// Generate a correlation ID
	correlationID := watermill.NewUUID()

	// Publish an event to initiate the signup process
	payload := discorduserevents.SignupStartedPayload{
		UserID:    r.UserID,
		ChannelID: userChannelID, // Using the correct channel ID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	msg := message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID) // Set the correlation ID in metadata

	// Publish the event to start the signup process
	if err := h.EventBus.Publish(discorduserevents.SignupStarted, msg); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing SignupStarted event", err)
		return
	}
}
