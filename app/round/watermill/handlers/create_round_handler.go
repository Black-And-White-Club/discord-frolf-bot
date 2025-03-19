package roundhandlers

import (
	"fmt"
	"log/slog"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error) {
	slog.Info("Handling round create requested", attr.CorrelationIDFromMsg(msg))
	var payload roundevents.CreateRoundRequestedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		slog.Error("Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		if err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(msg.Context(), msg.Metadata.Get("correlation_id"), "Failed to unmarshal payload: "+err.Error()); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Directly publish to the backend without additional checks
	backendMsg, err := h.Helpers.CreateResultMessage(msg, payload, roundevents.RoundCreateRequest)
	if err != nil {
		slog.Error("Failed to create result message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		if updateErr := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(msg.Context(), msg.Metadata.Get("correlation_id"), "Failed to create result message: "+err.Error()); updateErr != nil {
			return nil, updateErr
		}
		return nil, nil
	}

	slog.Info("Successfully processed round create request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

// Handles the RoundCreated Event from the Backend
func (h *RoundHandlers) HandleRoundCreated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	slog.Info("Handling round created event", attr.CorrelationIDFromMsg(msg))

	// Unmarshal the payload
	var payload roundevents.RoundCreatedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		slog.Error("Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Extract required data
	correlationID := msg.Metadata.Get("correlation_id")
	roundID := payload.RoundID
	channelID := "1344376922888474625"

	// 1️⃣ Update the original interaction response
	successMessage := fmt.Sprintf("✅ Round created successfully! Round ID: %d", roundID)
	if err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponse(ctx, correlationID, successMessage); err != nil {
		slog.Error("Failed to update interaction response", attr.Error(err))
		return nil, err
	}

	// 2️⃣ Send the embedded RSVP message with buttons
	description := ""
	if payload.Description != nil {
		description = string(*payload.Description)
	}
	location := ""
	if payload.Location != nil {
		location = string(*payload.Location)
	}

	discordMsg, err := h.RoundDiscord.GetCreateRoundManager().SendRoundEventEmbed(
		channelID,
		roundtypes.Title(payload.Title),
		roundtypes.Description(description),
		roundtypes.StartTime(*payload.StartTime),
		roundtypes.Location(location),
		roundtypes.UserID(payload.UserID),
		roundtypes.ID(roundID),
	)
	if err != nil {
		slog.Error("Failed to send round event embed", attr.Error(err))
		return nil, err
	}

	// 3️⃣ Publish the message ID as an event
	eventPayload := roundevents.RoundEventMessageIDUpdatedPayload{
		RoundID:        roundID,
		EventMessageID: discordMsg.ID,
	}
	resultMsg, err := h.Helpers.CreateResultMessage(msg, eventPayload, roundevents.RoundEventMessageIDUpdate)
	if err != nil {
		slog.Error("Failed to create result message", attr.Error(err))
		return nil, err
	}

	slog.Info("Successfully processed round embed", attr.CorrelationIDFromMsg(msg))

	return []*message.Message{resultMsg}, nil
}

func (h *RoundHandlers) HandleRoundCreationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	slog.Info("Handling round creation failed event", attr.CorrelationIDFromMsg(msg))
	var payload discordroundevents.RoundCreationFailedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		slog.Error("Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	correlationID := msg.Metadata.Get("correlation_id")

	// Prepare the error message
	errorMessage := "❌ Round creation failed: " + payload.Reason

	// Call the gateway handler to update the interaction response with a retry button
	if err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage); err != nil {
		slog.Error("Failed to update interaction response", attr.Error(err))
		return nil, err
	}
	return nil, nil
}

func (h *RoundHandlers) HandleRoundValidationFailed(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	slog.Info("Received round validation failed message", attr.CorrelationIDFromMsg(msg))
	var payload roundevents.RoundValidationFailedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		slog.Error("Failed to unmarshal payload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	correlationID := msg.Metadata.Get("correlation_id")

	errorMessages := payload.ErrorMessage
	errorMessage := "❌ " + strings.Join(errorMessages, "\n") + " Please try again."
	slog.Warn("Round validation failed", attr.UserID(string(payload.UserID)), attr.String("error", errorMessage))

	if err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage); err != nil {
		slog.Error("Failed to update interaction response", attr.Error(err))
		return nil, err
	}

	slog.Info("Successfully handled round validation failure", attr.CorrelationIDFromMsg(msg))

	return nil, nil
}
