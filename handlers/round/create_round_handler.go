package roundhandlers

import (
	"encoding/json"
	"fmt"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round create requested", attr.CorrelationIDFromMsg(msg))

	var payload discordroundevents.CreateRoundRequestedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// --- Domain/Business Logic Validation ---
	if time.Now().After(payload.StartTime) {
		h.Logger.Warn(ctx, "Start time is in the past", attr.CorrelationIDFromMsg(msg), attr.UserID(payload.UserID))
		errorPayload := discordroundevents.RoundCreationFailedPayload{
			UserID: payload.UserID,
			Reason: "Start time must be in the future.",
		}
		errorMsg, err := h.createResultMessage(msg, errorPayload, discordroundevents.RoundCreationFailedTopic) // Use helper
		if err != nil {
			return nil, err // createResultMessage logs
		}
		return []*message.Message{errorMsg}, nil // Ack
	}

	if payload.EndTime.Before(payload.StartTime) {
		h.Logger.Warn(ctx, "End time is before start time", attr.CorrelationIDFromMsg(msg), attr.UserID(payload.UserID))
		errorPayload := discordroundevents.RoundCreationFailedPayload{
			UserID: payload.UserID,
			Reason: "End time must be after start time.",
		}
		errorMsg, err := h.createResultMessage(msg, errorPayload, discordroundevents.RoundCreationFailedTopic) // Use helper
		if err != nil {
			return nil, err
		}
		return []*message.Message{errorMsg}, nil // Ack
	}

	// --- Construct Backend Payload ---
	backendPayload := roundevents.RoundCreateRequestPayload{
		Title:       payload.Title,
		Description: &payload.Description,
		StartTime:   payload.StartTime,
		EndTime:     payload.EndTime,
		Location:    &payload.Location,
		UserID:      payload.UserID,
		ChannelID:   payload.ChannelID, // Include ChannelID
	}
	backendMsg, err := h.createResultMessage(msg, backendPayload, roundevents.RoundCreateRequestTopic) // Use helper
	if err != nil {
		return nil, err
	}

	// --- Return the message for the router to publish ---
	h.Logger.Info(ctx, "Successfully processed round create request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

func (h *RoundHandlers) HandleRoundCreated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round created event", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.RoundCreatedEventPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	internalPayload := &discordroundevents.RoundCreatedPayload{
		RoundID:     payload.RoundID,
		Title:       payload.Title,
		StartTime:   payload.StartTime,
		EndTime:     payload.EndTime,
		RequesterID: payload.UserID,
		ChannelID:   payload.ChannelID,
	}
	internalMsg, err := h.createResultMessage(msg, internalPayload, discordroundevents.RoundCreatedTopic) // Use helper
	if err != nil {
		return nil, err
	}
	h.Logger.Info(ctx, "Successfully processed round created event", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{internalMsg}, nil
}
