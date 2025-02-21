package roundhandlers

import (
	"encoding/json"
	"fmt"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/en"
)

// HandleRoundTitleCollected handles the round title input event.
func (h *RoundHandlers) HandleRoundTitleCollected(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Prompt user for round title
	h.sendUserMessage(userID, "Please enter the round title:", ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateCollectingTitle
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundTitleCollected, nil, msg, ctx, nil)
}

// HandleRoundTitleResponse processes the user's response for the round title.
func (h *RoundHandlers) HandleRoundTitleResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please enter the round title:", ctx.CorrelationID)
		return nil
	}

	ctx.Title = response
	ctx.State = discordroundevents.StateCollectingTitle
	if err := h.setCache(userID, ctx); err != nil {
		return err
	}

	// Proceed to the next step
	return h.publishEvent(userID, discordroundevents.RoundTitleResponse, nil, msg, ctx, map[string]string{"title": response})
}

// HandleRoundDescriptionCollected prompts the user for the round description.
func (h *RoundHandlers) HandleRoundDescriptionCollected(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Prompt for round description
	h.sendUserMessage(userID, "Please enter the round description:", ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateCollectingDescription
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundDescriptionCollected, nil, msg, ctx, nil)
}

// HandleRoundDescriptionResponse processes the user's response for the round description.
func (h *RoundHandlers) HandleRoundDescriptionResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please enter the round description:", ctx.CorrelationID)
		return nil
	}

	ctx.Description = response
	ctx.State = discordroundevents.StateCollectingDescription
	if err := h.setCache(userID, ctx); err != nil {
		return err
	}

	// Proceed to the next step
	return h.publishEvent(userID, discordroundevents.RoundDescriptionResponse, nil, msg, ctx, map[string]string{"description": response})
}

// HandleRoundStartTimeCollected prompts the user for the round start time.
func (h *RoundHandlers) HandleRoundStartTimeCollected(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Prompt for round start time with examples
	h.sendUserMessage(userID, "Please enter the round start time (e.g., 'tomorrow at 3pm', 'tonight at 11:10 pm', 'next Wednesday at 2:25 p.m'):", ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateCollectingStartTime
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundStartTimeCollected, nil, msg, ctx, nil)
}

// HandleRoundStartTimeResponse processes the user's response for the round start time.
func (h *RoundHandlers) HandleRoundStartTimeResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please enter the round start time (e.g., 'tomorrow at 3pm', 'tonight at 11:10 pm', 'next Wednesday at 2:25 p.m'):", ctx.CorrelationID)
		return nil
	}

	// Initialize the when package
	w := when.New(nil)
	w.Add(en.All...)
	result, err := w.Parse(response, time.Now())
	if err != nil || result == nil {
		h.sendUserMessage(userID, "Invalid start time format. Please enter the start time in a natural language format (e.g., 'tomorrow at 3pm', 'tonight at 11:10 pm', 'next Wednesday at 2:25 p.m'):", ctx.CorrelationID)
		return nil
	}

	// Store the start time in UTC
	ctx.StartTime = result.Time.UTC()
	ctx.State = discordroundevents.StateCollectingEndTime
	if err := h.setCache(userID, ctx); err != nil {
		return err
	}

	// Proceed to the next step
	return h.publishEvent(userID, discordroundevents.RoundStartTimeResponse, nil, msg, ctx, map[string]string{"startTime": ctx.StartTime.Format(time.RFC3339)})
}

// HandleRoundEndTimeCollected prompts the user for the round end time.
func (h *RoundHandlers) HandleRoundEndTimeCollected(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Prompt for round end time with examples
	h.sendUserMessage(userID, "Please enter the round end time (e.g., '3 hours later', 'next Tuesday at 14:00') or type 'skip' to default to 3 hours:", ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateCollectingEndTime
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundEndTimeCollected, nil, msg, ctx, nil)
}

// HandleRoundEndTimeResponse processes the user's response for the round end time.
func (h *RoundHandlers) HandleRoundEndTimeResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please enter the round end time (e.g., '3 hours later', 'next Tuesday at 14:00') or type 'skip' to skip:", ctx.CorrelationID)
		return nil
	}

	if response == "skip" {
		ctx.EndTime = ctx.StartTime.Add(3 * time.Hour).UTC() // Automatically add 3 hours to the start time and store in UTC
		ctx.State = discordroundevents.StateCollectingLocation
		if err := h.setCache(userID, ctx); err != nil {
			return err
		}

		// Proceed to the next step
		return h.publishEvent(userID, discordroundevents.RoundEndTimeResponse, nil, msg, ctx, map[string]string{"endTime": ctx.EndTime.Format(time.RFC3339)})
	}

	// Initialize the when package
	w := when.New(nil)
	w.Add(en.All...)
	result, err := w.Parse(response, ctx.StartTime)
	if err != nil || result == nil {
		h.sendUserMessage(userID, "Invalid end time format. Please enter the end time in a natural language format (e.g., '3 hours later', 'next Tuesday at 14:00'):", ctx.CorrelationID)
		return nil
	}

	// Store the end time in UTC
	ctx.EndTime = result.Time.UTC()
	ctx.State = discordroundevents.StateCollectingLocation
	if err := h.setCache(userID, ctx); err != nil {
		return err
	}

	// Proceed to the next step
	return h.publishEvent(userID, discordroundevents.RoundEndTimeResponse, nil, msg, ctx, map[string]string{"endTime": ctx.EndTime.Format(time.RFC3339)})
}

// HandleRoundLocationCollected prompts the user for the round location.
func (h *RoundHandlers) HandleRoundLocationCollected(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Prompt for round location
	h.sendUserMessage(userID, "Please enter the round location:", ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateCollectingLocation
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundLocationCollected, nil, msg, ctx, nil)
}

// HandleRoundLocationResponse processes the user's response for the round location.
func (h *RoundHandlers) HandleRoundLocationResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please enter the round location:", ctx.CorrelationID)
		return nil
	}

	ctx.Location = response
	ctx.State = discordroundevents.StateCollectingLocation
	if err := h.setCache(userID, ctx); err != nil {
		return err
	}

	return h.publishEvent(userID, discordroundevents.RoundLocationResponse, nil, msg, ctx, map[string]string{"location": response})
}

// HandleRoundConfirmationRequest prompts the user to confirm the round details.
func (h *RoundHandlers) HandleRoundConfirmationRequest(msg *message.Message) error {
	userID, ctx, err := h.getUserContextFromMessage(msg)
	if err != nil {
		return err
	}

	// Format the round details
	roundDetails := fmt.Sprintf("Please confirm the round details:\nTitle: %s\nStart Time: %s\nLocation: %s\nType 'confirm' to proceed, 'back' to go back a step, 'cancel' to cancel, or 'restart' to start over.",
		ctx.Title, ctx.StartTime.Format(time.RFC3339), ctx.Location)

	// Prompt for confirmation
	h.sendUserMessage(userID, roundDetails, ctx.CorrelationID)

	// Set a timeout for the response
	ctx.State = discordroundevents.StateConfirmation
	if err := h.setCache(userID, ctx, 5*time.Minute); err != nil {
		return err
	}

	// Publish waiting event
	return h.publishEvent(userID, discordroundevents.RoundConfirmationRequest, nil, msg, ctx, nil)
}

// HandleRoundConfirmationResponse processes the user's confirmation response.
func (h *RoundHandlers) HandleRoundConfirmationResponse(msg *message.Message) error {
	userID, ctx, response, err := h.getUserResponseFromMessage(msg)
	if err != nil {
		return err
	}

	if response == "" {
		h.sendUserMessage(userID, "No response received. Please confirm the round details.", ctx.CorrelationID)
		return nil
	}

	h.handleUserResponse(userID, response, ctx)

	return h.publishEvent(ctx.CorrelationID, discordroundevents.RoundCreateRequest, nil, msg, ctx, map[string]string{
		"title":       ctx.Title,
		"startTime":   ctx.StartTime.Format(time.RFC3339),
		"endTime":     ctx.EndTime.Format(time.RFC3339),
		"description": ctx.Description,
		"location":    ctx.Location,
	})
}

// getUserContextFromMessage retrieves user context from cache based on message payload.
func (h *RoundHandlers) getUserContextFromMessage(msg *message.Message) (string, *discordroundevents.RoundCreationContext, error) {
	var payload discordroundevents.RoundEventPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling message payload: %w", err)
	}

	userID := payload.UserID

	ctx, err := h.getRoundCreationContext(userID)
	if err != nil {
		return "", nil, fmt.Errorf("error getting round creation context: %w", err)
	}

	return userID, ctx, nil
}
