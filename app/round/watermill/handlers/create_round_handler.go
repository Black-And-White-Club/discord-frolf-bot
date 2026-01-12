package roundhandlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

func (h *RoundHandlers) HandleRoundCreateRequested(ctx context.Context, payload *sharedroundevents.CreateRoundModalPayloadV1) ([]handlerwrapper.Result, error) {
	// Convert to backend payload and set GuildID
	backendPayload := roundevents.CreateRoundRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID(payload.GuildID),
		Title:       payload.Title,
		Description: payload.Description,
		StartTime:   payload.StartTime,
		Location:    payload.Location,
		UserID:      payload.UserID,
		ChannelID:   payload.ChannelID,
		Timezone:    payload.Timezone,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundCreationRequestedV1,
			Payload: backendPayload,
			Metadata: map[string]string{
				"submitted_at":    time.Now().UTC().Format(time.RFC3339),
				"user_timezone":   string(payload.Timezone),
				"raw_start_time":  payload.StartTime,
			},
		},
	}, nil
}

// Handles the RoundCreated Event from the Backend
func (h *RoundHandlers) HandleRoundCreated(ctx context.Context, payload *roundevents.RoundCreatedPayloadV1) ([]handlerwrapper.Result, error) {
	roundID := payload.RoundID
	guildID := string(payload.GuildID)
	channelID := string(payload.ChannelID)

	description := ""
	if payload.Description != nil {
		description = string(*payload.Description)
	}
	location := ""
	if payload.Location != nil {
		location = string(*payload.Location)
	}

	sendResult, err := h.RoundDiscord.GetCreateRoundManager().SendRoundEventEmbed(
		guildID,
		channelID,
		roundtypes.Title(payload.Title),
		roundtypes.Description(description),
		sharedtypes.StartTime(*payload.StartTime),
		roundtypes.Location(location),
		sharedtypes.DiscordID(payload.UserID),
		roundID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send round event embed: %w", err)
	}

	discordMsg, ok := sendResult.Success.(*discordgo.Message)
	if !ok || discordMsg == nil {
		if sendResult.Error != nil {
			return nil, fmt.Errorf("SendRoundEventEmbed service returned failure: %w", sendResult.Error)
		}
		return nil, fmt.Errorf("SendRoundEventEmbed did not return a Discord message on success for round %s", roundID.String())
	}

	discordMessageID := discordMsg.ID

	// Ensure GuildID is always set in the payload
	finalGuildID := payload.GuildID
	if finalGuildID == "" {
		return nil, fmt.Errorf("GuildID missing in payload for round %s", roundID.String())
	}

	updatePayload := roundevents.RoundMessageIDUpdatePayloadV1{
		GuildID: finalGuildID,
		RoundID: roundID,
	}

	return []handlerwrapper.Result{
		{
			Topic:   roundevents.RoundEventMessageIDUpdateV1,
			Payload: updatePayload,
			Metadata: map[string]string{
				"message_id": discordMessageID,
			},
		},
	}, nil
}

func (h *RoundHandlers) HandleRoundCreationFailed(ctx context.Context, payload *roundevents.RoundCreationFailedPayloadV1) ([]handlerwrapper.Result, error) {
	// Prepare the error message
	errorMessage := "❌ Round creation failed: " + payload.ErrorMessage

	// Call the gateway handler to update the interaction response with a retry button
	// This is a side-effect only, no outgoing messages
	correlationID, ok := ctx.Value("correlation_id").(string)
	if !ok {
		// correlationID not available in context, skip update
		return nil, nil
	}

	_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to update interaction response: %w", err)
	}

	return nil, nil
}

func (h *RoundHandlers) HandleRoundValidationFailed(ctx context.Context, payload *roundevents.RoundValidationFailedPayloadV1) ([]handlerwrapper.Result, error) {
	errorMessages := payload.ErrorMessages
	errorMessage := "❌ " + strings.Join(errorMessages, "\n") + " Please try again."

	correlationID, ok := ctx.Value("correlation_id").(string)
	if !ok {
		// correlationID not available in context, skip update
		return nil, nil
	}

	_, err := h.RoundDiscord.GetCreateRoundManager().UpdateInteractionResponseWithRetryButton(ctx, correlationID, errorMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to update interaction response: %w", err)
	}

	return nil, nil
}
