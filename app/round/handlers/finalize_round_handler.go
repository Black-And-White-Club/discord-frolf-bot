package handlers

import (
	"context"
	"fmt"

	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundFinalized handles the DiscordRoundFinalized event and updates the Discord embed
func (h *RoundHandlers) HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedDiscordPayloadV1) ([]handlerwrapper.Result, error) {
	discordChannelID := payload.DiscordChannelID
	if discordChannelID == "" {
		if h.guildConfigResolver != nil {
			guildCfg, err := h.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
			if err != nil || guildCfg == nil {
				h.logger.WarnContext(ctx, "failed to resolve guild config for round finalized, falling back to global config",
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

	// Get message ID from context (set by wrapper from message metadata).
	// Backfill rounds have no pre-existing Discord message, so an empty ID is valid.
	discordMessageID, _ := ctx.Value("discord_message_id").(string)

	// Convert the Discord-specific payload into the embed update payload
	// expected by the FinalizeScorecardEmbed manager.
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

	finalizeRoundManager := h.service.GetFinalizeRoundManager()

	var finalizeResult finalizeround.FinalizeRoundOperationResult
	var err error

	if discordMessageID == "" {
		// Backfill path: no existing Discord message — POST a new finalized embed.
		finalizeResult, err = finalizeRoundManager.PostFinalizedEmbed(ctx, discordChannelID, embedPayload)
	} else {
		finalizeResult, err = finalizeRoundManager.FinalizeScorecardEmbed(
			ctx,
			discordMessageID,
			discordChannelID,
			embedPayload,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to finalize scorecard embed: %w", err)
	}

	if finalizeResult.Error != nil {
		return nil, fmt.Errorf("finalize scorecard embed operation failed: %w", finalizeResult.Error)
	}

	// Set the native Discord Scheduled Event to COMPLETED (best-effort).
	if payload.DiscordEventID != "" {
		session := h.service.GetSession()
		_, err = session.GuildScheduledEventEdit(string(payload.GuildID), payload.DiscordEventID, &discordgo.GuildScheduledEventParams{
			Status: discordgo.GuildScheduledEventStatusCompleted,
		})
		if err != nil {
			h.logger.WarnContext(ctx, "failed to set native event to COMPLETED",
				"discord_event_id", payload.DiscordEventID,
				"error", err,
			)
		}
	}

	// We intentionally do not emit a trace event here. Returning result
	// messages causes Watermill to attempt publishing; if the trace topic
	// has no configured consumer/stream, publish will fail and the input
	// message will be Nacked, retrying the handler and duplicating side
	// effects. Returning an empty result set avoids that.
	return []handlerwrapper.Result{}, nil
}
