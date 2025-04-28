package roundhandlers

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundCreateRequested",
		&roundevents.CreateRoundRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createPayload := payload.(*roundevents.CreateRoundRequestedPayload)

			// Directly publish to the backend without additional checks
			backendMsg, err := h.Helpers.CreateResultMessage(msg, createPayload, roundevents.RoundCreateRequest)
			if err != nil {
				_, updateErr := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, msg.Metadata.Get("correlation_id"), "Failed to create result message: "+err.Error())
				if updateErr != nil {
					return nil, updateErr
				}
				return nil, nil
			}

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// Handles the RoundCreated Event from the Backend
func (h *RoundHandlers) HandleRoundCreated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundCreated",
		&roundevents.RoundCreatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createdPayload := payload.(*roundevents.RoundCreatedPayload)

			// Extract required data
			correlationID := msg.Metadata.Get("correlation_id")
			roundID := createdPayload.RoundID
			channelID := "1344376922888474625"

			// 1️⃣ Update the original interaction response
			successMessage := fmt.Sprintf("✅ Round created successfully! Round ID: %s", roundID)
			_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponse(ctx, correlationID, successMessage)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
				return nil, err
			}

			// 2️⃣ Send the embedded RSVP message with buttons
			description := ""
			if createdPayload.Description != nil {
				description = string(*createdPayload.Description)
			}
			location := ""
			if createdPayload.Location != nil {
				location = string(*createdPayload.Location)
			}

			result, err := h.RoundDiscord.GetCreateRoundManager().SendRoundEventEmbed(
				channelID,
				roundtypes.Title(createdPayload.Title),
				roundtypes.Description(description),
				sharedtypes.StartTime(*createdPayload.StartTime),
				roundtypes.Location(location),
				sharedtypes.DiscordID(createdPayload.UserID),
				sharedtypes.RoundID(roundID),
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to send round event embed", attr.Error(err))
				return nil, err
			}

			// Check if the operation was successful but result contains an error
			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Error in result from SendRoundEventEmbed", attr.Error(result.Error))
				return nil, result.Error
			}

			// Use the same roundID as the eventMessageID since they should be the same
			eventMessageID := sharedtypes.RoundID(roundID)

			// Log that we're using roundID as eventMessageID
			h.Logger.InfoContext(ctx, "Using round ID as event message ID",
				attr.RoundID("roundID", roundID),
				attr.RoundID("eventMessageID", eventMessageID))

			// 3️⃣ Publish the message ID as an event
			eventPayload := roundevents.RoundEventMessageIDUpdatedPayload{
				RoundID:        roundID,
				EventMessageID: eventMessageID,
			}

			resultMsg, err := h.Helpers.CreateResultMessage(msg, eventPayload, roundevents.RoundEventMessageIDUpdate)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err))
				return nil, err
			}

			return []*message.Message{resultMsg}, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundCreationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundCreationFailed",
		&discordroundevents.RoundCreationFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failedPayload := payload.(*discordroundevents.RoundCreationFailedPayload)
			correlationID := msg.Metadata.Get("correlation_id")

			// Prepare the error message
			errorMessage := "❌ Round creation failed: " + failedPayload.Reason

			// Call the gateway handler to update the interaction response with a retry button
			_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
				return nil, err
			}
			return nil, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundValidationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundValidationFailed",
		&roundevents.RoundValidationFailedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			validationPayload := payload.(*roundevents.RoundValidationFailedPayload)
			correlationID := msg.Metadata.Get("correlation_id")

			errorMessages := validationPayload.ErrorMessage
			errorMessage := "❌ " + strings.Join(errorMessages, "\n") + " Please try again."
			h.Logger.WarnContext(ctx, "Round validation failed",
				attr.UserID(validationPayload.UserID),
				attr.String("error", errorMessage))

			_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update interaction response", attr.Error(err))
				return nil, err
			}

			return nil, nil
		},
	)(msg)
}
