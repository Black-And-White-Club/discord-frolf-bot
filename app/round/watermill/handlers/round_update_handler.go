package roundhandlers

import (
	"context"
	"fmt"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundUpdateRequested(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundUpdateRequested",
		&discordroundevents.DiscordRoundUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			discordPayload := payload.(*discordroundevents.DiscordRoundUpdateRequestPayload)

			// Convert to backend payload and set GuildID
			var startTimeStr *string
			if discordPayload.StartTime != nil {
				timeValue := time.Time(*discordPayload.StartTime)
				timeStr := timeValue.Format("2006-01-02T15:04:05Z07:00")
				startTimeStr = &timeStr
			}

			backendPayload := roundevents.UpdateRoundRequestedPayload{
				GuildID:     sharedtypes.GuildID(discordPayload.GuildID),
				RoundID:     discordPayload.RoundID,
				UserID:      discordPayload.UserID,
				ChannelID:   discordPayload.ChannelID,
				MessageID:   discordPayload.MessageID,
				Title:       discordPayload.Title,
				Description: discordPayload.Description,
				StartTime:   startTimeStr,
				Location:    discordPayload.Location,
				// Note: Timezone field may need to be added if available in Discord payload
			}

			// Extract Discord metadata from the original payload
			channelID := discordPayload.ChannelID
			messageID := discordPayload.MessageID
			userID := string(discordPayload.UserID)

			h.Logger.InfoContext(ctx, "DEBUG: Backend received UpdateRoundRequestedPayload",
				attr.RoundID("received_round_id", discordPayload.RoundID),
				attr.String("received_round_id_string", discordPayload.RoundID.String()),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID),
				attr.Any("raw_payload", discordPayload))

			// ✅ Add this debug to verify the payload has message_id
			h.Logger.InfoContext(ctx, "DEBUG: Payload message_id from modal",
				attr.String("payload_message_id", discordPayload.MessageID),
				attr.String("payload_channel_id", discordPayload.ChannelID))

			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, roundevents.RoundUpdateRequest)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err))
				return nil, err
			}

			// Initialize metadata if nil
			if backendMsg.Metadata == nil {
				backendMsg.Metadata = message.Metadata{}
			}

			// Preserve Discord metadata in the outgoing message
			backendMsg.Metadata.Set("channel_id", channelID)
			backendMsg.Metadata.Set("message_id", messageID)
			backendMsg.Metadata.Set("user_id", userID)

			h.Logger.InfoContext(ctx, "DEBUG: Sending message to backend with metadata",
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID))

			// ✅ Add this debug to see what's actually in the metadata
			h.Logger.InfoContext(ctx, "DEBUG: All metadata being sent to backend")
			for key, value := range backendMsg.Metadata {
				h.Logger.InfoContext(ctx, "DEBUG: Metadata",
					attr.String("key", key),
					attr.String("value", value))
			}

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundUpdated",
		&roundevents.RoundEntityUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatedPayload := payload.(*roundevents.RoundEntityUpdatedPayload)

			// Extract Discord metadata from message metadata
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			// Extract round data from the payload
			round := updatedPayload.Round
			roundID := round.ID

			h.Logger.InfoContext(ctx, "Handling round updated event",
				attr.RoundID("round_id", roundID),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID))

			// ✅ Log all metadata for debugging
			h.Logger.InfoContext(ctx, "DEBUG: All metadata received in HandleRoundUpdated")
			for key, value := range msg.Metadata {
				h.Logger.InfoContext(ctx, "DEBUG: Metadata",
					attr.String("key", key),
					attr.String("value", value))
			}

			// Extract updated fields from round
			title := &round.Title

			// Handle pointer fields correctly
			var description *roundtypes.Description
			if round.Description != nil { // Check if pointer is not nil
				description = round.Description // Already a pointer, just assign
			}

			var startTime *sharedtypes.StartTime
			if round.StartTime != nil { // Check if pointer is not nil
				startTime = round.StartTime // Already a pointer, just assign
			}

			var location *roundtypes.Location
			if round.Location != nil { // Check if pointer is not nil
				location = round.Location // Already a pointer, just assign
			}

			// Validate required fields
			if channelID == "" {
				err := fmt.Errorf("channel ID is required for updating round embed")
				h.Logger.ErrorContext(ctx, "Missing channel ID - TEMPORARILY ACKING TO DEBUG", attr.Error(err))
				// ✅ Return success instead of error to stop retries
				return []*message.Message{}, nil
			}

			if messageID == "" {
				err := fmt.Errorf("message ID is required for updating round embed")
				h.Logger.ErrorContext(ctx, "Missing message ID - TEMPORARILY ACKING TO DEBUG", attr.Error(err))
				// ✅ Return success instead of error to stop retries
				return []*message.Message{}, nil
			}

			result, err := h.RoundDiscord.GetUpdateRoundManager().UpdateRoundEventEmbed(
				ctx,
				channelID,
				messageID,
				title,
				description,
				startTime,
				location,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed",
					attr.Error(err),
					attr.RoundID("round_id", roundID),
					attr.String("message_id", messageID))
				return nil, err
			}

			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Error in result from UpdateRoundEventEmbed",
					attr.Error(result.Error),
					attr.RoundID("round_id", roundID))
				return nil, result.Error
			}

			h.Logger.InfoContext(ctx, "Successfully updated round event embed",
				attr.RoundID("round_id", roundID),
				attr.String("message_id", messageID))

			return nil, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundUpdateFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundUpdateFailed",
		&roundevents.RoundUpdateErrorPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			failedPayload := payload.(*roundevents.RoundUpdateErrorPayload)

			// Log the error
			h.Logger.ErrorContext(ctx, "Round update failed", attr.String("error", failedPayload.Error))

			return nil, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundUpdateValidationFailed(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundUpdateValidationFailed",
		&roundevents.RoundUpdateValidatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			validatedPayload := payload.(*roundevents.RoundUpdateValidatedPayload)

			h.Logger.InfoContext(ctx, "Round update validated", attr.RoundID("round_id", validatedPayload.RoundUpdateRequestPayload.RoundID))

			return nil, nil
		},
	)(msg)
}
