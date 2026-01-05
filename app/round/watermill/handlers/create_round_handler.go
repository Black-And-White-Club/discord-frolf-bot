package roundhandlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func (h *RoundHandlers) HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error) {
	if h.Logger != nil {
		h.Logger.Info("HandleRoundCreateRequested invoked", attr.String("message_id", msg.UUID))
	}
	return h.handlerWrapper(
		"HandleRoundCreateRequested",
		&sharedroundevents.CreateRoundModalPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Check context health at start of handler logic
			if err := ctx.Err(); err != nil {
				h.Logger.ErrorContext(ctx, "Context cancelled at start of HandleRoundCreateRequested",
					attr.String("message_id", msg.UUID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("context cancelled: %w", err)
			}

			discordPayload := payload.(*sharedroundevents.CreateRoundModalPayloadV1)

			if h.Logger != nil {
				h.Logger.InfoContext(ctx, "HandleRoundCreateRequested processing payload", attr.Any("payload", discordPayload))
			}

			// Convert to backend payload and set GuildID
			backendPayload := roundevents.CreateRoundRequestedPayloadV1{
				GuildID:     sharedtypes.GuildID(discordPayload.GuildID),
				Title:       discordPayload.Title,
				Description: discordPayload.Description,
				StartTime:   discordPayload.StartTime,
				Location:    discordPayload.Location,
				UserID:      discordPayload.UserID,
				ChannelID:   discordPayload.ChannelID,
				Timezone:    discordPayload.Timezone,
			}

			if h.Logger != nil {
				h.Logger.InfoContext(ctx, "HandleRoundCreateRequested publishing backendPayload", attr.Any("backendPayload", backendPayload))
			}

			// Directly publish to the backend without additional checks
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, roundevents.RoundCreationRequestedV1)
			if err != nil {
				_, updateErr := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, msg.Metadata.Get("correlation_id"), "Failed to create result message: "+err.Error())
				if updateErr != nil {
					return nil, updateErr
				}
				return nil, nil
			}

			// Anchor metadata for deterministic relative time parsing downstream
			// Ensure metadata map initialized to prevent assignment to nil map panics in tests
			if backendMsg.Metadata == nil {
				backendMsg.Metadata = make(message.Metadata)
			}
			backendMsg.Metadata.Set("submitted_at", time.Now().UTC().Format(time.RFC3339))
			backendMsg.Metadata.Set("user_timezone", string(discordPayload.Timezone))
			backendMsg.Metadata.Set("raw_start_time", discordPayload.StartTime)

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// Handles the RoundCreated Event from the Backend
func (h *RoundHandlers) HandleRoundCreated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundCreated",
		&roundevents.RoundCreatedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createdPayload := payload.(*roundevents.RoundCreatedPayloadV1)

			h.Logger.InfoContext(ctx, "Received RoundCreated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", createdPayload.RoundID),
				attr.String("title", string(createdPayload.Title)),
				attr.String("user_id", string(createdPayload.UserID)),
			)

			roundID := createdPayload.RoundID
			guildID := string(createdPayload.GuildID)
			channelID := string(createdPayload.ChannelID)
			h.Logger.InfoContext(ctx, "Using channel ID for embed send", attr.String("channel_id", channelID))

			interactionCorrelationID, ok := msg.Metadata["interaction_correlation_id"]
			if ok && interactionCorrelationID != "" {
				successMessage := fmt.Sprintf("✅ Round created successfully! Round ID: %s", roundID)
				_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponse(ctx, interactionCorrelationID, successMessage)
				if err != nil {
					h.Logger.ErrorContext(ctx, "Failed to update original interaction response",
						attr.CorrelationIDFromMsg(msg),
						attr.String("interaction_correlation_id", interactionCorrelationID),
						attr.Error(err))
				}
			} else {
				h.Logger.WarnContext(ctx, "Interaction correlation ID not found or empty in metadata, skipping response update",
					attr.CorrelationIDFromMsg(msg),
				)
			}

			description := ""
			if createdPayload.Description != nil {
				description = string(*createdPayload.Description)
			}
			location := ""
			if createdPayload.Location != nil {
				location = string(*createdPayload.Location)
			}

			sendResult, err := h.RoundDiscord.GetCreateRoundManager().SendRoundEventEmbed(
				guildID,
				channelID,
				roundtypes.Title(createdPayload.Title),
				roundtypes.Description(description),
				sharedtypes.StartTime(*createdPayload.StartTime),
				roundtypes.Location(location),
				sharedtypes.DiscordID(createdPayload.UserID),
				roundID,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed during SendRoundEventEmbed service call",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to send round event embed: %w", err)
			}

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

			discordMessageID := discordMsg.ID
			h.Logger.InfoContext(ctx, "Successfully sent Discord embed message and captured ID",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
				attr.String("channel_id", discordMsg.ChannelID),
			)

			// --- PATCH: Ensure GuildID is always set in the payload ---
			finalGuildID := createdPayload.GuildID
			if finalGuildID == "" {
				if metaGuildID, ok := msg.Metadata["guild_id"]; ok && metaGuildID != "" {
					finalGuildID = sharedtypes.GuildID(metaGuildID)
					h.Logger.WarnContext(ctx, "GuildID was empty in payload, using metadata fallback",
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", string(finalGuildID)),
					)
				} else {
					h.Logger.ErrorContext(ctx, "GuildID missing in both payload and metadata",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundID),
					)
				}
			}

			updatePayload := roundevents.RoundMessageIDUpdatePayloadV1{
				GuildID: finalGuildID,
				RoundID: roundID,
			}

			tempMsgWithMetadata := &message.Message{
				Metadata: make(message.Metadata),
			}
			for key, val := range msg.Metadata {
				tempMsgWithMetadata.Metadata[key] = val
			}
			tempMsgWithMetadata.Metadata["discord_message_id"] = discordMessageID

			resultMsg, err := h.Helpers.CreateResultMessage(tempMsgWithMetadata, updatePayload, roundevents.RoundEventMessageIDUpdateV1)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create RoundEventMessageIDUpdate message",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create result message: %w", err)
			}

			h.Logger.InfoContext(ctx, "Publishing RoundEventMessageIDUpdate event with Discord message ID in metadata",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
			)

			return []*message.Message{resultMsg}, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundCreationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundCreationFailed",
		&roundevents.RoundCreationFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failedPayload := payload.(*roundevents.RoundCreationFailedPayloadV1)
			correlationID := msg.Metadata.Get("correlation_id")

			// Prepare the error message
			errorMessage := "❌ Round creation failed: " + failedPayload.ErrorMessage

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
		&roundevents.RoundValidationFailedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			validationPayload := payload.(*roundevents.RoundValidationFailedPayloadV1)
			correlationID := msg.Metadata.Get("correlation_id")

			errorMessages := validationPayload.ErrorMessages
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
