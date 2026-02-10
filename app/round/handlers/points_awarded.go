package handlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandlePointsAwarded handles the PointsAwardedV1 event.
// It retrieves the cached round payload, updates participant points, and refreshes the embed.
func (h *RoundHandlers) HandlePointsAwarded(ctx context.Context, payload *sharedevents.PointsAwardedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling points awarded event",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("round_id", payload.RoundID.String()),
		attr.Int("player_count", len(payload.Points)),
	)

	// Retrieve cached payload
	cacheKey := fmt.Sprintf("round_payload:%s", payload.RoundID)
	cached, err := h.interactionStore.Get(ctx, cacheKey)
	if err != nil {
		h.logger.WarnContext(ctx, "round payload not found in cache or expired", "round_id", payload.RoundID, "error", err)
		return []handlerwrapper.Result{}, nil // Return nil error to avoid retries for expired cache
	}

	embedPayload, ok := cached.(roundevents.RoundFinalizedEmbedUpdatePayloadV1)
	if !ok {
		return nil, fmt.Errorf("unexpected type in cache for round_payload:%s", payload.RoundID)
	}

	// Update participants with points
	updatedCount := 0
	for userID, points := range payload.Points {
		for i := range embedPayload.Participants {
			if embedPayload.Participants[i].UserID == userID {
				p := points // copy
				embedPayload.Participants[i].Points = &p
				updatedCount++
				break
			}
		}
	}

	if updatedCount == 0 {
		h.logger.WarnContext(ctx, "no participants updated with points", "round_id", payload.RoundID)
		return []handlerwrapper.Result{}, nil
	}

	// Get message ID from context (set by wrapper from message metadata)
	discordMessageID, ok := ctx.Value("discord_message_id").(string)
	if !ok || discordMessageID == "" {
		// If points are awarded via a background process, we might lack the message ID in context.
		// However, RoundFinalized should have just finished and points are usually awarded immediately after.
		// If it's missing, we use the EventMessageID from the payload if available.
		discordMessageID = string(embedPayload.EventMessageID)
	}

	if discordMessageID == "" {
		return nil, fmt.Errorf("missing discord_message_id for points awarding update")
	}

	discordChannelID := h.config.GetEventChannelID()

	// Update the embed
	finalizeRoundManager := h.service.GetFinalizeRoundManager()
	finalizeResult, err := finalizeRoundManager.FinalizeScorecardEmbed(
		ctx,
		discordMessageID,
		discordChannelID,
		embedPayload,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update scorecard embed with points: %w", err)
	}

	if finalizeResult.Error != nil {
		return nil, fmt.Errorf("finalize scorecard embed update failed: %w", finalizeResult.Error)
	}

	// Clean up cache (optional, but good practice)
	h.interactionStore.Delete(ctx, cacheKey)

	return []handlerwrapper.Result{}, nil
}
