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
// It updates participant points on the scorecard embed.
func (h *RoundHandlers) HandlePointsAwarded(ctx context.Context, payload *sharedevents.PointsAwardedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling points awarded event",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("round_id", payload.RoundID.String()),
		attr.Int("player_count", len(payload.Points)),
	)

	// Validate essential enriched data
	if payload.EventMessageID == "" {
		// If enrichment failed or event is from legacy backend, we can't update the embed without the ID.
		// However, we can try to fall back if the context has it, but enrichment is the primary path now.
		// Use context ID as fallback if available
		if msgID, ok := ctx.Value("discord_message_id").(string); ok && msgID != "" {
			payload.EventMessageID = msgID
		} else {
			h.logger.WarnContext(ctx, "skipping points display update: missing event_message_id in payload", "round_id", payload.RoundID)
			return []handlerwrapper.Result{}, nil
		}
	}

	// Early return if no points to apply — avoids an unnecessary Discord API call
	if len(payload.Points) == 0 {
		h.logger.InfoContext(ctx, "no points in payload, skipping embed update", "round_id", payload.RoundID)
		return []handlerwrapper.Result{}, nil
	}

	// Construct the embed update payload directly from the enriched event
	embedPayload := roundevents.RoundFinalizedEmbedUpdatePayloadV1{
		GuildID:          payload.GuildID,
		RoundID:          payload.RoundID,
		Title:            payload.Title,
		StartTime:        payload.StartTime,
		Location:         payload.Location,
		Participants:     payload.Participants,
		Teams:            payload.Teams,
		EventMessageID:   payload.EventMessageID,
		DiscordChannelID: payload.DiscordChannelID,
	}

	// Ensure participants have the correct points from the payload map
	// The backend enrichment might have already set them, but we enforce consistency with the map
	for userID, points := range payload.Points {
		for i := range embedPayload.Participants {
			if embedPayload.Participants[i].UserID == userID {
				p := points
				embedPayload.Participants[i].Points = &p
				break
			}
		}
	}

	discordChannelID := payload.DiscordChannelID
	if discordChannelID == "" {
		if h.guildConfigResolver != nil {
			guildCfg, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
			if err != nil || guildCfg == nil {
				h.logger.WarnContext(ctx, "failed to resolve guild config for points update, falling back to global config",
					attr.String("guild_id", string(payload.GuildID)),
					attr.Error(err))
				discordChannelID = h.config.GetEventChannelID()
			} else {
				discordChannelID = guildCfg.EventChannelID
			}
		} else {
			discordChannelID = h.config.GetEventChannelID()
		}
	}

	// Update the embed
	finalizeRoundManager := h.service.GetFinalizeRoundManager()
	finalizeResult, err := finalizeRoundManager.FinalizeScorecardEmbed(
		ctx,
		embedPayload.EventMessageID,
		discordChannelID,
		embedPayload,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update scorecard embed with points: %w", err)
	}

	if finalizeResult.Error != nil {
		return nil, fmt.Errorf("finalize scorecard embed update failed: %w", finalizeResult.Error)
	}

	return []handlerwrapper.Result{}, nil
}
