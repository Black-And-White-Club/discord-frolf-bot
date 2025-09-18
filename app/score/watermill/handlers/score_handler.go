package scorehandlers

import (
	"context"
	"fmt"
	"strings"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleScoreUpdateRequest passes Discord user score update to backend using shared payload.
func (h *ScoreHandlers) HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScoreUpdateRequest",
		&scoreevents.ScoreUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			req := payload.(*scoreevents.ScoreUpdateRequestPayload)

			if req.RoundID == sharedtypes.RoundID(uuid.Nil) || req.UserID == sharedtypes.DiscordID("") || req.Score == sharedtypes.Score(0) {
				h.Logger.ErrorContext(ctx, "Invalid ScoreUpdateRequest payload",
					attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("invalid payload: missing round_id, user_id, or score")
			}

			backendMsg, err := h.Helper.CreateResultMessage(msg, req, scoreevents.ScoreUpdateRequest)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend score update message",
					attr.Error(err),
					attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("failed to create backend score update message: %w", err)
			}

			// Preserve metadata for response routing
			backendMsg.Metadata.Set("user_id", string(req.UserID))
			backendMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
			backendMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}

// HandleScoreUpdateSuccess sends confirmation to Discord that score update succeeded.
func (h *ScoreHandlers) HandleScoreUpdateSuccess(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScoreUpdateSuccess",
		&scoreevents.ScoreUpdateSuccessPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			success := payload.(*scoreevents.ScoreUpdateSuccessPayload)

			// Extract metadata to route Discord response
			userID := msg.Metadata.Get("user_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			if userID == "" || channelID == "" {
				return nil, fmt.Errorf("missing routing metadata for Discord message")
			}

			resp := map[string]interface{}{
				"type":       "score_update_success",
				"user_id":    userID,
				"round_id":   success.RoundID,
				"score":      success.Score,
				"message_id": messageID,
			}

			discordMsg, err := h.Helper.CreateResultMessage(msg, resp, scoreevents.ScoreUpdateSuccess)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create Discord success message", attr.Error(err))
				return nil, fmt.Errorf("failed to create Discord success message: %w", err)
			}

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}

// HandleScoreUpdateFailure sends an error message to Discord when score update fails.
func (h *ScoreHandlers) HandleScoreUpdateFailure(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleScoreUpdateFailure",
		&scoreevents.ScoreUpdateFailurePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			fail := payload.(*scoreevents.ScoreUpdateFailurePayload)

			userID := msg.Metadata.Get("user_id")
			channelID := msg.Metadata.Get("channel_id")
			messageID := msg.Metadata.Get("message_id")

			// Always suppress retries and do NOT post Discord messages for the known business failure
			// where the aggregate scores row is missing. This prevents spam on redelivery.
			if strings.Contains(fail.Error, "score record not found") {
				h.Logger.InfoContext(ctx, "Suppressing retry for known business failure (aggregate scores missing)",
					attr.RoundID("round_id", fail.RoundID),
					attr.String("guild_id", string(fail.GuildID)),
					attr.String("user_id", string(fail.UserID)),
				)
				return nil, nil // ACK with no downstream messages
			}

			if userID == "" || channelID == "" {
				return nil, fmt.Errorf("missing routing metadata for Discord message")
			}

			resp := map[string]interface{}{
				"type":       "score_update_failure",
				"user_id":    userID,
				"round_id":   fail.RoundID,
				"error":      fail.Error,
				"message_id": messageID,
			}

			discordMsg, err := h.Helper.CreateResultMessage(msg, resp, scoreevents.ScoreUpdateFailure)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create Discord failure message", attr.Error(err))
				return nil, fmt.Errorf("failed to create Discord failure message: %w", err)
			}

			return []*message.Message{discordMsg}, nil
		},
	)(msg)
}
