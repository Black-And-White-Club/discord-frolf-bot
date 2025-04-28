package roundhandlers

import (
	"context"

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
			updatePayload := payload.(*discordroundevents.DiscordRoundUpdateRequestPayload)

			backendMsg, err := h.Helpers.CreateResultMessage(msg, updatePayload, roundevents.RoundUpdateRequest)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err))
				return nil, err
			}

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

func (h *RoundHandlers) HandleRoundUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundUpdated",
		&discordroundevents.DiscordRoundUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatedPayload := payload.(*discordroundevents.DiscordRoundUpdatedPayload)

			// Extract required data
			roundID := updatedPayload.RoundID
			channelID := updatedPayload.ChannelID

			// Update the embedded RSVP message
			title := updatedPayload.Title

			var description *roundtypes.Description
			if updatedPayload.Description != nil {
				desc := roundtypes.Description(*updatedPayload.Description)
				description = &desc
			}

			var startTime *sharedtypes.StartTime
			if updatedPayload.StartTime != nil {
				startTimeValue := sharedtypes.StartTime(*updatedPayload.StartTime)
				startTime = &startTimeValue
			}

			location := updatedPayload.Location

			result, err := h.RoundDiscord.GetUpdateRoundManager().UpdateRoundEventEmbed(
				ctx,
				channelID,
				roundID,
				title,
				description,
				startTime,
				location,
			)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to update round event embed", attr.Error(err))
				return nil, err
			}

			if result.Error != nil {
				h.Logger.ErrorContext(ctx, "Error in result from UpdateRoundEventEmbed", attr.Error(result.Error))
				return nil, result.Error
			}

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
