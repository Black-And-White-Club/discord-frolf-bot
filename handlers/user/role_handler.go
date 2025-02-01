package userhandlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleRoleUpdateCommand handles the role update command.
func (h *UserHandlers) HandleRoleUpdateCommand(s discord.Discord, m *discord.MessageCreate, wm *message.Message) {
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
		h.Logger.Info("Ignoring role update command from a guild", "user_id", m.Author.ID, "guild_id", m.GuildID)
		return
	}

	// Reply with a list of possible roles and the option to cancel
	roles := []usertypes.UserRoleEnum{
		usertypes.UserRoleRattler,
		usertypes.UserRoleEditor,
		usertypes.UserRoleAdmin,
	}

	roleOptions := "Please choose a role or type 'cancel' to cancel the request:\n"
	for _, role := range roles {
		roleOptions += string(role) + "\n"
	}
	roleOptions += "Cancel\n"

	h.sendUserMessage(s, m.Author.ID, roleOptions, correlationID)

	err = h.Cache.Set(correlationID, []byte(m.Author.ID))
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error setting cache", err)
		return
	}

	timeout := time.AfterFunc(3*time.Minute, func() {
		h.handleRoleRequestTimeout(s, correlationID)
	})
	defer timeout.Stop()

	h.Logger.Info("Publishing role request event", "user_id", m.Author.ID, "correlation_id", correlationID)

	event := message.NewMessage(correlationID, []byte(m.Author.ID))
	h.EventUtil.PropagateMetadata(wm, event)
	if err := h.EventBus.Publish(discorduserevents.RoleSelectRequest, event); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing role request event", err)
		return
	}
}

// HandleRoleResponse processes the user's response with their role selection.
func (h *UserHandlers) HandleRoleResponse(s discord.Discord, m *discord.MessageCreate, wm *message.Message) {
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
	role := m.Content
	if role == "cancel" {
		h.Logger.Info("User canceled role request", "user_id", userID)
		h.Cache.Delete(correlationID)
		h.sendUserMessage(s, userID, "Role request has been canceled.", correlationID)
		h.publishCancelEvent(correlationID, userID)
		return
	}

	roles := []usertypes.UserRoleEnum{
		usertypes.UserRoleRattler,
		usertypes.UserRoleEditor,
		usertypes.UserRoleAdmin,
	}

	validRoles := make(map[string]bool)
	for _, r := range roles {
		validRoles[string(r)] = true
	}

	if !validRoles[role] {
		h.ErrorReporter.ReportError(correlationID, "invalid role provided", fmt.Errorf("invalid role"), "user_id", userID, "role", role)
		h.sendUserMessage(s, userID, fmt.Sprintf("Invalid role. Please select %s, %s, or %s.", usertypes.UserRoleRattler, usertypes.UserRoleEditor, usertypes.UserRoleAdmin), correlationID)
		h.publishTraceEvent(correlationID, fmt.Sprintf("User provided invalid role: %s", role))
		return
	}

	h.Logger.Info("User provided role", "user_id", userID, "role", role)

	// Use JSON payload instead of plain strings
	payload := struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}{
		UserID: userID,
		Role:   role,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	event := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(wm, event)

	// Publish event & delete cache entry
	if err := h.EventBus.Publish(userevents.UserRoleUpdateRequest, event); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing event", err)
		return
	}
	h.Cache.Delete(correlationID) // Clean up after handling response
}

// handleRoleRequestTimeout handles the timeout for role requests.
func (h *UserHandlers) handleRoleRequestTimeout(s discord.Discord, correlationID string) {
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

	h.sendUserMessage(s, userID, "Your role request has timed out. Please try again.", correlationID)

	// Log timeout & publish event for traceability
	h.Logger.Error("Role request timed out", "user_id", userID, "correlationID", correlationID)
	h.publishTraceEvent(correlationID, fmt.Sprintf("Role request timed out for user: %s", userID))

	// Delete cache entry after timeout
	h.Cache.Delete(correlationID)
}

// HandleRoleUpdateResponse handles the backend's response to the role selection.
func (h *UserHandlers) HandleRoleUpdateResponse(s discord.Discord, msg *message.Message) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	topic := msg.Metadata.Get("topic")

	switch topic {
	case userevents.UserRoleUpdated:
		var payload userevents.UserRoleUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal UserRoleUpdated event: %w", err)
		}
		h.Logger.Info("Received UserRoleUpdated event", "correlation_id", correlationID, "user_id", payload.DiscordID, "role", payload.Role)

		channel, err := h.Session.UserChannelCreate(string(payload.DiscordID))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error creating user channel", err)
			return err
		}

		_, err = h.Session.ChannelMessageSend(channel.ID, fmt.Sprintf("Your role has been updated to %s.", payload.Role))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error sending message", err)
			return err
		}

	case userevents.UserRoleUpdateFailed:
		var payload userevents.UserRoleUpdateFailedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal UserRoleUpdateFailed event: %w", err)
		}
		h.Logger.Info("Received UserRoleUpdateFailed event", "correlation_id", correlationID, "user_id", payload.DiscordID, "reason", payload.Reason)

		channel, err := h.Session.UserChannelCreate(string(payload.DiscordID))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error creating user channel", err)
			return err
		}

		_, err = h.Session.ChannelMessageSend(channel.ID, fmt.Sprintf("Failed to update your role: %s. Please try again.", payload.Reason))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error sending message", err)
			return err
		}

	default:
		h.Logger.Error("unknown topic", "topic", topic)
		return fmt.Errorf("unknown topic: %s", topic)
	}

	// Publish an event for traceability
	traceEvent := message.NewMessage(correlationID, []byte(fmt.Sprintf("Role update response for user %s", correlationID)))
	h.EventUtil.PropagateMetadata(msg, traceEvent)
	h.EventBus.Publish(discorduserevents.RoleUpdateTrace, traceEvent)

	return nil
}

// publishRoleUpdateRequest publishes a UserRoleUpdateRequest event.
func (h *UserHandlers) publishRoleUpdateRequest(userID, role, correlationID string, srcMsg *message.Message) {
	payload := userevents.UserRoleUpdateRequestPayload{
		DiscordID:   usertypes.DiscordID(userID),
		Role:        usertypes.UserRoleEnum(role),
		RequesterID: userID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	// Publish the role update request event
	msg := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(srcMsg, msg)
	if err := h.EventBus.Publish(userevents.UserRoleUpdateRequest, msg); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing UserRoleUpdateRequest event", err)
		return
	}

	h.Logger.Info("role update request published", "correlation_id", correlationID, "user_id", userID, "role", role)
}

// publishRoleUpdateTimeoutEvent publishes an event indicating that the role update request timed out.
func (h *UserHandlers) publishRoleUpdateTimeoutEvent(userID, correlationID string, srcMsg *message.Message) {
	payload := discorduserevents.RoleUpdateTimeoutPayload{
		UserID: userID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	msg := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(srcMsg, msg)
	if err := h.EventBus.Publish(discorduserevents.RoleUpdateTimeout, msg); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing RoleUpdateTimeout event", err)
	}
}
