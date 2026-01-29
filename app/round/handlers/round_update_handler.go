package handlers

import (
	"context"
	"fmt"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

func (h *RoundHandlers) HandleRoundUpdateRequested(ctx context.Context, payload *discordroundevents.RoundUpdateModalSubmittedPayloadV1) ([]handlerwrapper.Result, error) {
	// Convert to backend payload and set GuildID
	var startTimeStr *string
	if payload.StartTime != nil {
		startTimeStr = payload.StartTime
	}

	backendPayload := roundevents.UpdateRoundRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(payload.GuildID),
		RoundID:     payload.RoundID,
		UserID:      payload.UserID,
		ChannelID:   payload.ChannelID,
		MessageID:   payload.MessageID,
		Title:       payload.Title,
		Description: payload.Description,
		StartTime:   startTimeStr,
		Timezone:    payload.Timezone,
		Location:    payload.Location,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundUpdateRequestedV1,
			Payload: backendPayload,
			Metadata: map[string]string{
				"discord_message_id": payload.MessageID,
				"channel_id":         payload.ChannelID,
				"message_id":         payload.MessageID,
				"user_id":            string(payload.UserID),
			},
		},
	}, nil
}

func (h *RoundHandlers) HandleRoundUpdated(ctx context.Context, payload *roundevents.RoundEntityUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Extract round data from the payload
	round := payload.Round

	// Extract channel ID and message ID from context (set by wrapper from message metadata)
	channelID, ok := ctx.Value("channel_id").(string)
	if !ok || channelID == "" {
		return nil, fmt.Errorf("channel ID is required for updating round embed")
	}

	messageID, ok := ctx.Value("message_id").(string)
	if !ok || messageID == "" {
		if discordMessageID, ok := ctx.Value("discord_message_id").(string); ok && discordMessageID != "" {
			messageID = discordMessageID
		} else {
			return nil, fmt.Errorf("message ID is required for updating round embed")
		}
	}

	// Extract updated fields from round
	var title *roundtypes.Title
	if round.Title != "" {
		title = &round.Title
	}

	// Handle optional fields by passing nil when not provided
	var description *roundtypes.Description
	if round.Description != "" {
		description = &round.Description
	}

	var startTime *sharedtypes.StartTime
	if round.StartTime != nil {
		startTime = round.StartTime
	}

	var location *roundtypes.Location
	if round.Location != "" {
		location = &round.Location
	}

	result, err := h.service.GetUpdateRoundManager().UpdateRoundEventEmbed(
		ctx,
		channelID,
		messageID,
		title,
		description,
		startTime,
		location,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update round event embed: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("update round event embed operation failed: %w", result.Error)
	}

	return nil, nil
}

func (h *RoundHandlers) HandleRoundUpdateFailed(ctx context.Context, payload *roundevents.RoundUpdateErrorPayloadV1) ([]handlerwrapper.Result, error) {
	// Pure domain logic - just return without side effects
	return nil, nil
}

func (h *RoundHandlers) HandleRoundUpdateValidationFailed(ctx context.Context, payload *roundevents.RoundUpdateValidatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Pure domain logic - just return without side effects
	return nil, nil
}
