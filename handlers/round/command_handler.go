package roundhandlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCreateRoundCommand handles the create round command.
func (h *RoundHandlers) HandleCreateRoundCommand(m *discord.MessageCreate, wm *message.Message) {
	// Generate a new correlation ID
	correlationID := watermill.NewUUID()
	combinedID := fmt.Sprintf("%s:%s", correlationID, m.Author.ID)

	// Get the bot user
	botUser, err := h.Session.GetBotUser()
	if err != nil {
		h.ErrorReporter.ReportError(combinedID, "error getting bot user", err)
		return
	}

	// Ignore messages from the bot itself
	if m.Author.ID == botUser.ID {
		return
	}

	// Initialize the round creation context
	ctx := &discordroundevents.RoundCreationContext{
		CorrelationID: combinedID,
		State:         discordroundevents.StateCollectingTitle,
		Title:         "",
		StartTime:     time.Time{},
		Location:      "",
		UserID:        m.Author.ID,
	}

	// Serialize the context to JSON
	ctxBytes, err := json.Marshal(ctx)
	if err != nil {
		h.ErrorReporter.ReportError(combinedID, "error marshaling context", err)
		return
	}

	// Store the context in BigCache
	h.Cache.Set(m.Author.ID, ctxBytes)

	// Publish event to start the round creation process
	payload := discordroundevents.RoundEventPayload{
		UserID: m.Author.ID,
	}
	if err := h.publishEvent(combinedID, discordroundevents.RoundStartCreation, payload, wm, ctx, nil); err != nil {
		h.ErrorReporter.ReportError(combinedID, "error publishing StartRoundCreation event", err)
		return
	}
}
