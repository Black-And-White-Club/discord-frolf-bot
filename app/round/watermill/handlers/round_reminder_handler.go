package roundhandlers

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleRoundReminder prepares and sends a round reminder notification.
func (h *RoundHandlers) HandleRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) ([]handlerwrapper.Result, error) {
	// Early validation - fail fast for invalid payloads
	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		return nil, nil // Don't return error - this payload is invalid and shouldn't be retried
	}

	// Use default channel from config if payload doesn't have one
	channelID := payload.DiscordChannelID
	if channelID == "" {
		defaultChannelID := h.config.GetEventChannelID()
		if defaultChannelID == "" {
			return nil, nil // Don't return error - configuration issue, no point retrying
		}
		channelID = defaultChannelID
		// Create new payload with default channel
		payload = &roundevents.DiscordReminderPayloadV1{
			RoundID:          payload.RoundID,
			DiscordChannelID: defaultChannelID,
			ReminderType:     payload.ReminderType,
			RoundTitle:       payload.RoundTitle,
			StartTime:        payload.StartTime,
			Location:         payload.Location,
			UserIDs:          payload.UserIDs,
			DiscordGuildID:   payload.DiscordGuildID,
			EventMessageID:   payload.EventMessageID,
		}
	}

	// Create timeout context for Discord API call (30 seconds max)
	apiCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send the Discord reminder with timeout protection
	result, err := h.service.GetRoundReminderManager().SendRoundReminder(apiCtx, payload)
	if err != nil {
		// Check if it's a timeout or context cancellation
		if apiCtx.Err() == context.DeadlineExceeded {
			return nil, nil // Don't retry timeouts - they'll likely timeout again
		}

		// For other errors, allow retry by returning error
		return nil, fmt.Errorf("failed to send round reminder: %w", err)
	}

	// Validate result with defensive check
	var success bool
	if successVal, ok := result.Success.(bool); ok {
		success = successVal
	}

	if !success {
		return nil, nil // Don't retry operational failures - they indicate the reminder was processed but failed
	}

	return nil, nil
}
