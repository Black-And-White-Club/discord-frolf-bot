package roundhandlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

// handleUserResponse processes the user's response based on predefined commands.
func (h *RoundHandlers) handleUserResponse(userID, response string, ctx *discordroundevents.RoundCreationContext) {
	switch response {
	case "cancel":
		h.cancelRoundCreation(userID, ctx)
	case "back":
		h.handleBack(userID, ctx)
	case "restart":
		h.restartRoundCreation(userID, ctx)
	case "confirm":
		h.handleConfirmation(userID, ctx)
	default:
		h.sendUserMessage(userID, "Invalid response. Please type 'confirm' to proceed, 'back' to go back a step, 'cancel' to cancel, or 'restart' to start over.", ctx.CorrelationID)
	}
}

// updateStateAndPreparePayload updates the state and prepares the event payload.
func (h *RoundHandlers) updateStateAndPreparePayload(userID string, ctx *discordroundevents.RoundCreationContext, newState string, additionalData map[string]string) (map[string]interface{}, error) {
	ctx.State = newState
	if err := h.setCache(userID, ctx); err != nil {
		return nil, fmt.Errorf("failed to update state in cache: %w", err)
	}

	payload := make(map[string]interface{})
	payload["userID"] = userID
	for key, value := range additionalData {
		payload[key] = value
	}

	return payload, nil
}

// handleConfirmation verifies inputs before sending the round creation request.
func (h *RoundHandlers) handleConfirmation(userID string, ctx *discordroundevents.RoundCreationContext) {
	if ctx.Title == "" || ctx.StartTime.IsZero() || ctx.Location == "" {
		h.sendUserMessage(userID, "Invalid input. Please ensure title, start time, and location are provided.", ctx.CorrelationID)
		return
	}

	h.sendUserMessage(userID, "Round creation confirmed. Creating round...", ctx.CorrelationID)

	// Publish event for round creation
	payload := map[string]interface{}{
		"userID":    userID,
		"title":     ctx.Title,
		"startTime": ctx.StartTime.Format(time.RFC3339),
		"location":  ctx.Location,
	}
	if err := h.publishEvent(ctx.CorrelationID, roundevents.RoundCreateRequest, payload, nil, nil, nil); err != nil {
		h.ErrorReporter.ReportError(ctx.CorrelationID, "Error publishing round confirmed event", err)
	}
}

// handleBack moves the user back one step in the process.
func (h *RoundHandlers) handleBack(userID string, ctx *discordroundevents.RoundCreationContext) {
	var stateOrder = []string{
		discordroundevents.StateCollectingTitle,
		discordroundevents.StateCollectingStartTime,
		discordroundevents.StateCollectingLocation,
		discordroundevents.StateConfirmation,
	}

	for i := range stateOrder {
		if stateOrder[i] == ctx.State && i > 0 {
			ctx.State = stateOrder[i-1]
			if err := h.setCache(userID, ctx); err != nil {
				h.ErrorReporter.ReportError(ctx.CorrelationID, "Failed to update state in cache", err)
				return
			}
			h.sendUserMessage(userID, "Please provide the previous step information again.", ctx.CorrelationID)
			// Dynamically determine the event to publish based on the new state
			eventMapping := map[string]string{
				discordroundevents.StateCollectingStartTime: discordroundevents.RoundEditTitle,
				discordroundevents.StateCollectingLocation:  discordroundevents.RoundEditStartTime,
				discordroundevents.StateConfirmation:        discordroundevents.RoundEditLocation,
			}
			if event, exists := eventMapping[ctx.State]; exists {
				if err := h.publishEvent(ctx.CorrelationID, event, nil, nil, ctx, nil); err != nil {
					h.ErrorReporter.ReportError(ctx.CorrelationID, "Error publishing next step event", err)
				}
			}
			return
		}
	}
}

// cancelRoundCreation terminates the round creation process.
func (h *RoundHandlers) cancelRoundCreation(userID string, ctx *discordroundevents.RoundCreationContext) {
	if err := h.Cache.Delete(userID); err != nil {
		h.ErrorReporter.ReportError(ctx.CorrelationID, "Failed to clear cache on round cancellation", err)
	}
	h.sendUserMessage(userID, "Round creation process has been canceled.", ctx.CorrelationID)
	h.publishCancelEvent(ctx.CorrelationID, userID)
}

// restartRoundCreation resets the round creation state.
func (h *RoundHandlers) restartRoundCreation(userID string, ctx *discordroundevents.RoundCreationContext) {
	ctx.State = discordroundevents.StateCollectingTitle
	ctx.Title, ctx.Location = "", ""
	ctx.StartTime = time.Time{}
	if err := h.setCache(userID, ctx); err != nil {
		h.ErrorReporter.ReportError(ctx.CorrelationID, "Failed to update state in cache", err)
		return
	}
	h.sendUserMessage(userID, "Round creation process has been restarted. Please enter the round title:", ctx.CorrelationID)
	if err := h.publishEvent(ctx.CorrelationID, discordroundevents.RoundStartCreation, discordroundevents.RoundEventPayload{UserID: userID}, nil, nil, nil); err != nil {
		h.ErrorReporter.ReportError(ctx.CorrelationID, "Error publishing start round creation event", err)
	}
}

// setCache stores the context in BigCache with proper error handling and write optimization.
// If a timeout is provided, it handles the timeout logic.
func (h *RoundHandlers) setCache(userID string, ctx *discordroundevents.RoundCreationContext, timeout ...time.Duration) error {
	existingData, err := h.Cache.Get(userID)
	if err == nil {
		var existingCtx discordroundevents.RoundCreationContext
		if json.Unmarshal(existingData, &existingCtx) == nil && existingCtx == *ctx {
			return nil // No changes, avoid unnecessary write
		}
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("error marshaling context for cache: %w", err)
	}

	// Store in cache
	if err := h.Cache.Set(userID, data); err != nil {
		return fmt.Errorf("error setting cache for userID %s: %w", userID, err)
	}

	// If a timeout is provided, handle the timeout logic
	if len(timeout) > 0 {
		go h.handleCacheTimeout(userID, ctx.CorrelationID, timeout[0])
	}

	return nil
}

// handleCacheTimeout handles the timeout for cache entries.
func (h *RoundHandlers) handleCacheTimeout(userID, correlationID string, timeout time.Duration) {
	time.Sleep(timeout)
	ctx, err := h.getRoundCreationContext(userID)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "error getting round creation context for timeout", err)
		return
	}

	if ctx.State != discordroundevents.StateConfirmation {
		h.sendUserMessage(userID, "Timeout reached. Please restart the round creation process.", correlationID)
		h.cancelRoundCreation(userID, ctx)
	}
}

// publishEvent publishes an event with the given payload and metadata.
// If ctx is provided, it updates the state and includes additional data in the payload.
func (h *RoundHandlers) publishEvent(correlationID, eventType string, payload interface{}, wm *message.Message, ctx *discordroundevents.RoundCreationContext, additionalData map[string]string) error {
	// If ctx is provided, update the state and prepare the payload
	if ctx != nil {
		var err error
		payload, err = h.updateStateAndPreparePayload(ctx.UserID, ctx, ctx.State, additionalData)
		if err != nil {
			return fmt.Errorf("error preparing payload for publishing: %w", err)
		}
	}

	// Marshal the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	// Create a new Watermill message
	msg := message.NewMessage(correlationID, payloadBytes)

	// Propagate metadata if provided
	if wm != nil {
		h.EventUtil.PropagateMetadata(wm, msg)
	}

	// Publish the event
	if err := h.EventBus.Publish(eventType, msg); err != nil {
		return fmt.Errorf("error publishing %s event: %w", eventType, err)
	}

	return nil
}

// getUserResponseFromMessage extracts the user's response from a message payload.
func (h *RoundHandlers) getUserResponseFromMessage(msg *message.Message) (string, *discordroundevents.RoundCreationContext, string, error) {
	var payload discordroundevents.RoundEventPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return "", nil, "", fmt.Errorf("error unmarshaling message payload: %w", err)
	}

	userID := payload.UserID
	response := payload.Response // Now correctly using the Response field

	ctx, err := h.getRoundCreationContext(userID)
	if err != nil {
		return "", nil, "", fmt.Errorf("error getting round creation context: %w", err)
	}

	return userID, ctx, response, nil
}

// getRoundCreationContext retrieves the round creation context from the cache.
func (h *RoundHandlers) getRoundCreationContext(userID string) (*discordroundevents.RoundCreationContext, error) {
	data, err := h.Cache.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting data from cache: %w", err)
	}

	var ctx discordroundevents.RoundCreationContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("error unmarshaling context from cache: %w", err)
	}

	return &ctx, nil
}

// extractUserID extracts the userID from the combined correlationID.
func extractUserID(combinedID string) (string, error) {
	parts := strings.Split(combinedID, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid combinedID format")
	}
	return parts[1], nil
}
