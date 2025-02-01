package userhandlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleAskIfUserHasTag asks the user if they have a tag number.
func (h *UserHandlers) HandleAskIfUserHasTag(s discord.Discord, wm *message.Message) {
	// Extract the correlation ID from the Watermill message metadata
	correlationID := wm.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Retrieve the user ID from the payload
	var payload discorduserevents.SignupStartedPayload
	if err := json.Unmarshal(wm.Payload, &payload); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error unmarshaling payload", err)
		return
	}
	userID := payload.UserID

	// Send the message to the user
	h.sendUserMessage(s, userID, "Do you have a tag number? (1. yes 2. no 3. cancel)", correlationID)

	// Save the state in the cache with a timeout
	err := h.Cache.Set(correlationID, []byte(userID))
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error setting cache", err)
		return
	}

	// Set a timeout to handle the case where the user does not respond
	timeout := time.AfterFunc(3*time.Minute, func() {
		h.handleSignupRequestTimeout(s, correlationID)
	})
	defer timeout.Stop()

	// Log the request event
	h.Logger.Info("Asking user if they have a tag number", "user_id", userID, "correlation_id", correlationID)

	// Publish an event to handle the user's response
	event := message.NewMessage(correlationID, wm.Payload)
	h.EventUtil.PropagateMetadata(wm, event)
	if err := h.EventBus.Publish(discorduserevents.SignupTagAsk, event); err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing SignupRequest event", err)
		return
	}
}

// HandleSignupRequest handles the user's response to the signup question.
func (h *UserHandlers) HandleSignupRequest(s discord.Discord, m *discord.MessageCreate, wm *message.Message) {
	// Extract the correlation ID from the Watermill message metadata
	correlationID := wm.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Retrieve the user ID from the cache
	userIDBytes, err := h.Cache.Get(correlationID)
	if err != nil || userIDBytes == nil {
		h.Logger.Info("Timeout skipped: User already responded", "correlationID", correlationID)
		return
	}
	userID := string(userIDBytes)

	// Process the user's response
	response := m.Content
	switch response {
	case "1", "yes":
		h.Logger.Info("User has a tag number", "user_id", userID)
		h.publishTagNumberRequestEvent(userID, correlationID, wm)
	case "2", "no":
		h.Logger.Info("User does not have a tag number", "user_id", userID)
		h.publishSignupRequestedEvent(userID, "", correlationID, wm)
	case "3", "cancel":
		h.Logger.Info("User canceled signup request", "user_id", userID)
		h.Cache.Delete(correlationID)
		h.sendUserMessage(s, userID, "Signup process has been canceled.", correlationID)
		h.publishCancelEvent(correlationID, userID)
	default:
		h.Logger.Error("invalid response provided", "user_id", userID, "response", response)
		h.sendUserMessage(s, userID, "Invalid response. Please respond with 1 for yes, 2 for no, or 3 to cancel.", correlationID)
		h.publishTraceEvent(correlationID, fmt.Sprintf("User provided invalid signup response: %s", response))
	}
}

// handleSignupRequestTimeout handles the timeout for the initial signup question.
func (h *UserHandlers) handleSignupRequestTimeout(s discord.Discord, correlationID string) {
	// Check if the user is still in the cache
	if _, err := h.Cache.Get(correlationID); err != nil {
		h.Logger.Info("Skipping timeout: User already responded", "correlationID", correlationID)
		return
	}

	userIDBytes, err := h.Cache.Get(correlationID)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error getting user ID from cache", err)
		return
	}
	userID := string(userIDBytes)

	h.sendUserMessage(s, userID, "Your signup request has timed out. Please try again.", correlationID)

	// Log timeout & publish event for traceability
	h.Logger.Error("Signup request timed out", "user_id", userID, "correlationID", correlationID)
	h.publishTraceEvent(correlationID, fmt.Sprintf("Signup request timed out for user: %s", userID))

	// Delete cache entry after timeout
	h.Cache.Delete(correlationID)
}

// publishTagNumberRequestEvent publishes a TagNumberRequested event.
func (h *UserHandlers) publishTagNumberRequestEvent(userID, correlationID string, srcMsg *message.Message) {
	payload := discorduserevents.TagNumberRequestedPayload{
		UserID: userID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	msg := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(srcMsg, msg)

	err = h.EventBus.Publish(discorduserevents.SignupTagIncludeRequested, msg)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing event", err)
		return
	}
}

// publishSignupRequestedEvent publishes a SignupRequested event.
func (h *UserHandlers) publishSignupRequestedEvent(userID, tagNumber, correlationID string, srcMsg *message.Message) {
	var tagNumberPtr *int
	if tagNumber != "" {
		tagNumberPtr = new(int)
		if tagNum, err := strconv.Atoi(tagNumber); err == nil {
			*tagNumberPtr = tagNum
		} else {
			h.ErrorReporter.ReportError(correlationID, "error converting tag number", err)
			return
		}
	} else {
		tagNumberPtr = nil
	}

	payload := userevents.UserSignupRequestPayload{
		DiscordID: usertypes.DiscordID(userID),
		TagNumber: tagNumberPtr,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error marshaling payload", err)
		return
	}

	msg := message.NewMessage(correlationID, payloadBytes)
	h.EventUtil.PropagateMetadata(srcMsg, msg)

	err = h.EventBus.Publish(userevents.UserSignupRequest, msg)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error publishing event", err)
		return
	}
}

// HandleSignupResponse handles the response from the backend indicating success or failure.
func (h *UserHandlers) HandleSignupResponse(msg *message.Message) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	topic := msg.Metadata.Get("topic")

	switch topic {
	case userevents.UserCreated:
		var userCreatedPayload userevents.UserCreatedPayload
		if err := json.Unmarshal(msg.Payload, &userCreatedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal UserCreated event: %w", err)
		}
		h.Logger.Info("Received UserCreated event", "correlation_id", correlationID, "user_id", userCreatedPayload.DiscordID)

		channel, err := h.Session.UserChannelCreate(string(userCreatedPayload.DiscordID))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error creating user channel", err)
			return err
		}

		_, err = h.Session.ChannelMessageSend(channel.ID, "Signup complete! You now have access to the #members-only channel.")
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error sending message", err)
			return err
		}

	case userevents.UserCreationFailed:
		var userCreationFailedPayload userevents.UserCreationFailedPayload
		if err := json.Unmarshal(msg.Payload, &userCreationFailedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal UserCreationFailed event: %w", err)
		}
		h.Logger.Info("Received UserCreationFailed event", "correlation_id", correlationID, "user_id", userCreationFailedPayload.DiscordID)

		channel, err := h.Session.UserChannelCreate(string(userCreationFailedPayload.DiscordID))
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error creating user channel", err)
			return err
		}

		_, err = h.Session.ChannelMessageSend(channel.ID, "Signup failed: "+userCreationFailedPayload.Reason+". Please try again.")
		if err != nil {
			h.ErrorReporter.ReportError(correlationID, "error sending message", err)
			return err
		}

	default:
		h.Logger.Error("unknown topic", "topic", topic)
		return fmt.Errorf("unknown topic: %s", topic)
	}

	// Publish an event for traceability
	traceEvent := message.NewMessage(correlationID, []byte(fmt.Sprintf("Signup response for user %s", correlationID)))
	h.EventUtil.PropagateMetadata(msg, traceEvent)
	h.EventBus.Publish(discorduserevents.SignupTrace, traceEvent)

	return nil
}
