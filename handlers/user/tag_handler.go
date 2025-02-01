package userhandlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleTagNumberRequest handles the initial request for the user's tag number.
func (h *UserHandlers) HandleIncludeTagNumberRequest(s discord.Discord, m *discord.MessageCreate, wm *message.Message) {
	correlationID := wm.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Get the bot user
	botUser, err := s.GetBotUser()
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error getting bot user", err)
		return
	}

	// Ignore messages from the bot itself
	if m.Author.ID == botUser.ID {
		return
	}

	// Ignore messages sent in guilds
	if m.GuildID != "" {
		h.Logger.Info("Ignoring tag number request from a guild", "user_id", m.Author.ID, "guild_id", m.GuildID)
		return
	}

	// Reply with a message asking for the tag number
	h.sendUserMessage(s, m.Author.ID, "Please provide your tag number or type 'cancel' to cancel the request.", correlationID)

	// Save the state in the cache with a timeout
	err = h.Cache.Set(correlationID, []byte(m.Author.ID))
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error setting cache", err)
		return
	}

	// Set a timeout to handle the case where the user does not respond
	timeout := time.AfterFunc(3*time.Minute, func() {
		h.handleTagNumberRequestTimeout(s, correlationID)
	})
	defer timeout.Stop()

	h.Logger.Info("Publishing tag number request event", "user_id", m.Author.ID, "correlation_id", correlationID)

	// Create and publish the tag number request event with correlation ID in metadata
	event := message.NewMessage(correlationID, []byte(m.Author.ID))
	h.EventUtil.PropagateMetadata(wm, event)
	if err := h.EventBus.Publish(discorduserevents.TagNumberRequested, event); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing tag request event", err)
		return
	}
}

// HandleTagNumberResponse processes the user's response with their tag number.
func (h *UserHandlers) HandleIncludeTagNumberResponse(s discord.Discord, m *discord.MessageCreate, wm *message.Message) {
	// Extract the correlation ID from the Watermill message metadata
	correlationID := wm.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Retrieve the user ID from the cache
	userIDBytes, err := h.Cache.Get(correlationID)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error getting user ID from cache", err)
		return
	}

	if userIDBytes == nil {
		h.Logger.Info("Skipping timeout: User already responded", "correlationID", correlationID)
		return
	}
	userID := string(userIDBytes)

	// Process the user's response
	tagNumber := m.Content
	if tagNumber == "cancel" {
		h.Logger.Info("User canceled tag number request", "user_id", userID)
		h.Cache.Delete(correlationID)
		h.sendUserMessage(s, userID, "Tag number request has been canceled.", correlationID)
		h.publishCancelEvent(correlationID, userID)
		return
	}

	if _, err := strconv.Atoi(tagNumber); err != nil {
		h.ErrorReporter.ReportError(correlationID, "invalid tag number provided", err, "user_id", userID, "tag_number", tagNumber)
		h.sendUserMessage(s, userID, "Invalid tag number. Please provide a valid integer tag number.", correlationID)
		return
	}

	h.Logger.Info("User provided tag number", "user_id", userID, "tag_number", tagNumber)

	// Use JSON payload instead of plain strings
	payload := struct {
		UserID    string `json:"user_id"`
		TagNumber string `json:"tag_number"`
	}{
		UserID:    userID,
		TagNumber: tagNumber,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	event := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(wm, event)

	// Publish event & delete cache entry
	if err := h.EventBus.Publish(discorduserevents.TagNumberResponse, event); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing event", err)
		return
	}
	h.Cache.Delete(correlationID) // Clean up after handling response
}

// handleTagNumberRequestTimeout handles the timeout for tag number requests.
func (h *UserHandlers) handleTagNumberRequestTimeout(s discord.Discord, correlationID string) {
	// Retrieve the user ID from the cache
	userIDBytes, err := h.Cache.Get(correlationID)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error getting user ID from cache", err)
		return
	}

	if userIDBytes == nil {
		h.Logger.Info("Skipping timeout: User already responded", "correlationID", correlationID)
		return
	}
	userID := string(userIDBytes)

	h.sendUserMessage(s, userID, "Your tag number request has timed out. Please try again.", correlationID)

	// Log timeout & publish event for traceability
	h.Logger.Error("Tag number request timed out", "user_id", userID, "correlationID", correlationID)
	h.publishTraceEvent(correlationID, fmt.Sprintf("Tag number request timed out for user: %s", userID))

	// Delete cache entry after timeout
	h.Cache.Delete(correlationID)
}
