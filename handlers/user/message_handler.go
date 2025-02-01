package userhandlers

import (
	"encoding/json"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleMessageCreate handles incoming Discord messages.
func (h *UserHandlers) HandleMessageCreate(s discord.Discord, m *discord.MessageCreate) {
	botUser, err := s.GetBotUser()
	if err != nil {
		h.ErrorReporter.ReportError("", "error getting bot user", err)
		return
	}

	// Ignore all messages from the bot itself
	if m.Author.ID == botUser.ID {
		return
	}

	// Ignore messages sent in guilds
	if m.GuildID != "" {
		return
	}

	// Get the channel ID. If there's an error, log and return.
	if _, err := h.getChannelID(s, m.Author.ID); err != nil {
		h.ErrorReporter.ReportError("", "error getting channel id", err, "user_id", m.Author.ID)
		return
	}

	content := strings.TrimSpace(strings.ToLower(m.Content))

	if strings.HasPrefix(content, "!updaterole") {
		// Generate a correlation ID
		correlationID := watermill.NewUUID()

		// Create the payload
		payload := struct {
			UserID  string `json:"user_id"`
			Content string `json:"content"`
		}{
			UserID:  m.Author.ID,
			Content: content,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
			return
		}

		// Create and publish the message
		msg := message.NewMessage(correlationID, payloadBytes)
		msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

		if err := h.EventBus.Publish(discorduserevents.RoleUpdateCommand, msg); err != nil {
			h.ErrorReporter.ReportError(correlationID, "error publishing role update command event", err)
			return
		}

		return
	}

	h.Logger.Info("unknown message type", "message", m.Content)
}
