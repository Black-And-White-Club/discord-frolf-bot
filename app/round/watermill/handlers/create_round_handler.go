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
	"github.com/bwmarrin/discordgo"
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

			h.Logger.InfoContext(ctx, "Received RoundCreated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", createdPayload.RoundID),
				attr.String("title", string(createdPayload.Title)),
				attr.String("user_id", string(createdPayload.UserID)),
			)

			// Extract required data
			roundID := createdPayload.RoundID
			// **Revert channelID back to the hardcoded value as requested**
			channelID := h.Config.Discord.EventChannelID // Hardcoded value

			// Find the correlation ID from the original interaction response if available
			// Use standard map lookup with ok check
			interactionCorrelationID, ok := msg.Metadata["interaction_correlation_id"]
			if ok && interactionCorrelationID != "" { // Check if key exists and value is not empty
				// 1️⃣ Update the original interaction response
				successMessage := fmt.Sprintf("✅ Round created successfully! Round ID: %s", roundID)
				// Pass the extracted interactionCorrelationID (string)
				_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponse(ctx, interactionCorrelationID, successMessage)
				if err != nil {
					h.Logger.ErrorContext(ctx, "Failed to update original interaction response",
						attr.CorrelationIDFromMsg(msg),
						attr.String("interaction_correlation_id", interactionCorrelationID),
						attr.Error(err))
					// Decide whether to return nil, err here or just log and continue
				}
			} else {
				h.Logger.WarnContext(ctx, "Interaction correlation ID not found or empty in metadata, skipping response update",
					attr.CorrelationIDFromMsg(msg),
				)
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

			// Call SendRoundEventEmbed
			sendResult, err := h.RoundDiscord.GetCreateRoundManager().SendRoundEventEmbed(
				channelID, // Use the hardcoded channel ID
				roundtypes.Title(createdPayload.Title),
				roundtypes.Description(description),
				sharedtypes.StartTime(*createdPayload.StartTime),
				roundtypes.Location(location),
				sharedtypes.DiscordID(createdPayload.UserID), // Creator's Discord ID
				roundID, // Pass the RoundID
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed during SendRoundEventEmbed service call",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to send round event embed: %w", err)
			}

			// Check if the operation was successful according to its result
			// Assuming CreateRoundOperationResult.Success holds *discordgo.Message on success
			discordMsg, ok := sendResult.Success.(*discordgo.Message)
			if !ok || discordMsg == nil {
				if sendResult.Error != nil {
					h.Logger.ErrorContext(ctx, "SendRoundEventEmbed service returned failure result",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundID),
						attr.Any("service_error", sendResult.Error),
					)
					return nil, fmt.Errorf("SendRoundEventEmbed service returned failure: %w", sendResult.Error)
				}

				h.Logger.ErrorContext(ctx, "SendRoundEventEmbed service returned non-message success result",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Any("send_result_success", sendResult.Success),
				)
				return nil, fmt.Errorf("SendRoundEventEmbed did not return a Discord message on success for round %s", roundID.String())
			}

			// **Extract the Discord message ID from the sent message result**
			discordMessageID := discordMsg.ID
			h.Logger.InfoContext(ctx, "Successfully sent Discord embed message and captured ID",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID), // Log the captured ID
				attr.String("channel_id", discordMsg.ChannelID),     // Log the channel ID from the message
			)

			// Create the payload for the update event
			updatePayload := struct { // Simple payload, ID goes in metadata
				RoundID sharedtypes.RoundID `json:"round_id"`
			}{
				RoundID: roundID,
			}

			tempMsgWithMetadata := &message.Message{
				Metadata: make(message.Metadata), // Create a new Metadata map
			}
			// Copy existing metadata from the original incoming message
			for key, val := range msg.Metadata {
				tempMsgWithMetadata.Metadata[key] = val // Standard map assignment
			}

			// **Set the captured Discord message ID in the temporary message's metadata**
			tempMsgWithMetadata.Metadata["discord_message_id"] = discordMessageID // **Standard map assignment**

			// Now use CreateResultMessage, passing the temporary message for metadata copying
			resultMsg, err := h.Helpers.CreateResultMessage(tempMsgWithMetadata, updatePayload, roundevents.RoundEventMessageIDUpdate)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create RoundEventMessageIDUpdate message",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.String("discord_message_id", discordMessageID), // Use captured ID in log
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create result message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Publishing RoundEventMessageIDUpdate event with Discord message ID in metadata",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID), // Log the ID from the message being published
			)

			// Return the message that publishes the Discord message ID update event
			return []*message.Message{resultMsg}, nil
		},
	)(msg) // Execute the wrapped handler
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
