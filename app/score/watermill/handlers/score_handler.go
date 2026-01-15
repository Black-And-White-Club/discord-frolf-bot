package scorehandlers

import (
	"context"
	"fmt"
	"strings"

	discordscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// Typed handlers for router-driven pure transformations.
// These return []handlerwrapper.Result and contain only domain logic.
func (h *ScoreHandlers) HandleScoreUpdateRequestTyped(ctx context.Context, payload *discordscoreevents.ScoreUpdateRequestDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload is nil")
	}

	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) || payload.UserID == sharedtypes.DiscordID("") || payload.Score == sharedtypes.Score(0) {
		return nil, fmt.Errorf("invalid payload: missing round_id, user_id, or score")
	}

	backendPayload := sharedevents.ScoreUpdateRequestedPayloadV1{
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		UserID:    payload.UserID,
		Score:     payload.Score,
		TagNumber: payload.TagNumber,
	}

	md := map[string]string{
		"user_id":    string(payload.UserID),
		"channel_id": payload.ChannelID,
		"message_id": payload.MessageID,
	}

	return []handlerwrapper.Result{{
		Topic:    sharedevents.ScoreUpdateRequestedV1,
		Payload:  backendPayload,
		Metadata: md,
	}}, nil
}

func (h *ScoreHandlers) HandleScoreUpdateSuccessTyped(ctx context.Context, payload *sharedevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload is nil")
	}

	// Build response payload for Discord
	resp := map[string]interface{}{
		"type":       "score_update_success",
		"round_id":   payload.RoundID,
		"score":      payload.Score,
		"message_id": "", // populated by metadata propagation in wrapper
	}

	return []handlerwrapper.Result{{
		Topic:   discordscoreevents.ScoreUpdateResponseDiscordV1,
		Payload: resp,
	}}, nil
}

func (h *ScoreHandlers) HandleScoreUpdateFailureTyped(ctx context.Context, payload *sharedevents.ScoreUpdateFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload is nil")
	}

	// Suppress known business failure to avoid duplicate Discord posts
	if strings.Contains(payload.Reason, "score record not found") {
		h.Logger.InfoContext(ctx, "Suppressing retry for known business failure (aggregate scores missing)",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.String("user_id", string(payload.UserID)),
		)
		return nil, nil
	}

	resp := map[string]interface{}{
		"type":       "score_update_failure",
		"round_id":   payload.RoundID,
		"error":      payload.Reason,
		"message_id": "",
	}

	return []handlerwrapper.Result{{
		Topic:   discordscoreevents.ScoreUpdateFailedDiscordV1,
		Payload: resp,
	}}, nil
}
