package handlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleParticipantScoreUpdated handles a successful participant score update event from the backend.
// It calls the scoreround.UpdateScoreEmbed function to update the scorecard.
func (h *RoundHandlers) HandleParticipantScoreUpdated(ctx context.Context, payload *roundevents.ParticipantScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Use ChannelID from config if payload ChannelID is empty (as a fallback)
	channelID := payload.ChannelID
	if channelID == "" && h.config != nil && h.config.GetEventChannelID() != "" {
		channelID = h.config.GetEventChannelID()
	}

	// Resolve messageID: prefer payload, fall back to MessageMap
	messageID := payload.EventMessageID
	if messageID == "" {
		if id, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			messageID = id
		}
	}

	// Get the ScoreRoundManager
	scoreRoundManager := h.service.GetScoreRoundManager()

	// Call UpdateScoreEmbed with the specific user's updated score
	updateResult, err := scoreRoundManager.UpdateScoreEmbed(
		ctx,
		channelID,      // Pass channel ID
		messageID,      // Pass message ID of the scorecard
		payload.UserID, // Pass the UserID of the updated participant
		&payload.Score, // Pass a pointer to the updated score
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call UpdateScoreEmbed: %w", err)
	}

	if updateResult.Error != nil {
		return nil, fmt.Errorf("scorecard update failed: %w", updateResult.Error)
	}

	return nil, nil
}

// HandleScoresBulkUpdated updates the scorecard for bulk score overrides.
func (h *RoundHandlers) HandleScoresBulkUpdated(ctx context.Context, payload *roundevents.RoundScoresBulkUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	// Use ChannelID from config if payload ChannelID is empty (as a fallback)
	channelID := payload.ChannelID
	if channelID == "" && h.config != nil && h.config.GetEventChannelID() != "" {
		channelID = h.config.GetEventChannelID()
	}

	// Resolve messageID: prefer payload, fall back to MessageMap
	messageID := payload.EventMessageID
	if messageID == "" {
		if id, found := h.service.GetMessageMap().Load(payload.RoundID); found {
			messageID = id
		}
	}

	h.logger.InfoContext(ctx, "HandleScoresBulkUpdated",
		"channel_id", channelID,
		"message_id", messageID,
		"participant_count", len(payload.Participants))

	scoreRoundManager := h.service.GetScoreRoundManager()
	updateResult, err := scoreRoundManager.UpdateScoreEmbedBulk(
		ctx,
		channelID,
		messageID,
		payload.Participants,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call UpdateScoreEmbedBulk: %w", err)
	}
	if updateResult.Error != nil {
		return nil, fmt.Errorf("bulk scorecard update failed: %w", updateResult.Error)
	}

	return nil, nil
}

// HandleScoreUpdateError processes a failed score update event.
func (h *RoundHandlers) HandleScoreUpdateError(ctx context.Context, payload *roundevents.RoundScoreUpdateErrorPayloadV1) ([]handlerwrapper.Result, error) {
	if payload.Error == "" {
		return nil, fmt.Errorf("received empty error message in HandleScoreUpdateError")
	}

	_, err := h.service.GetScoreRoundManager().SendScoreUpdateError(ctx, payload.ScoreUpdateRequest.UserID, payload.Error)
	if err != nil {
		return nil, fmt.Errorf("failed to send score update error notification: %w", err)
	}

	// No downstream messages are required for score update errors in this module;
	// return nil so nothing is published. (Trace events are not consumed anywhere.)
	return nil, nil
}
