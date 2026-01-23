package finalizeround

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

// FinalizeScorecardEmbed updates the round embed when a round is finalized
func (frm *finalizeRoundManager) FinalizeScorecardEmbed(
	ctx context.Context,
	eventMessageID string,
	channelID string,
	embedPayload roundevents.RoundFinalizedEmbedUpdatePayloadV1,
) (FinalizeRoundOperationResult, error) {

	return frm.operationWrapper(ctx, "FinalizeScorecardEmbed", func(ctx context.Context) (FinalizeRoundOperationResult, error) {
		if frm.session == nil {
			return FinalizeRoundOperationResult{}, fmt.Errorf("discord session is nil")
		}

		if eventMessageID == "" {
			return FinalizeRoundOperationResult{}, fmt.Errorf("event message ID is empty")
		}

		// Resolve guild ID
		guildID := ""
		if embedPayload.GuildID != "" {
			guildID = string(embedPayload.GuildID)
		}

		// Resolve channel ID if not explicitly provided
		resolvedChannelID := channelID
		if resolvedChannelID == "" && frm.guildConfigResolver != nil && guildID != "" {
			if cfg, err := frm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
				frm.logger.InfoContext(ctx, "Resolved channel ID from guild config",
					attr.String("channel_id", resolvedChannelID),
					attr.String("guild_id", guildID),
				)
			}
		}

		if resolvedChannelID == "" {
			return FinalizeRoundOperationResult{}, fmt.Errorf("channel ID could not be resolved")
		}

		// Fetch existing message to preserve fields like Location
		existingMsg, err := frm.session.ChannelMessage(resolvedChannelID, eventMessageID)
		if err != nil {
			return FinalizeRoundOperationResult{}, fmt.Errorf("failed to fetch existing message: %w", err)
		}

		// Extract original location from existing embed if payload is missing it
		originalLocation := extractLocationFromMessage(existingMsg)

		if (embedPayload.Location == "" || string(embedPayload.Location) == "") && originalLocation != "" {
			loc := roundtypes.Location(originalLocation)
			embedPayload.Location = loc

			frm.logger.InfoContext(ctx, "Preserved original location from existing embed",
				attr.String("location", originalLocation),
				attr.RoundID("round_id", embedPayload.RoundID),
			)
		}

		// Transform payload into finalized embed
		embed, components, err := frm.TransformRoundToFinalizedScorecard(embedPayload)
		if err != nil {
			return FinalizeRoundOperationResult{}, fmt.Errorf("failed to transform finalized scorecard: %w", err)
		}

		if embed == nil {
			return FinalizeRoundOperationResult{}, fmt.Errorf("transformed embed is nil")
		}

		edit := &discordgo.MessageEdit{
			Channel: resolvedChannelID,
			ID:      eventMessageID,
			Embeds:  &[]*discordgo.MessageEmbed{embed},
		}

		// Explicitly set components only when provided
		if components != nil {
			edit.Components = &components
		}

		updatedMsg, err := frm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return FinalizeRoundOperationResult{}, fmt.Errorf("failed to edit finalized embed: %w", err)
		}

		frm.logger.InfoContext(ctx, "Successfully finalized round embed on Discord",
			attr.String("discord_message_id", eventMessageID),
			attr.String("channel_id", resolvedChannelID),
			attr.RoundID("round_id", embedPayload.RoundID),
		)

		return FinalizeRoundOperationResult{Success: updatedMsg}, nil
	})
}

func extractLocationFromMessage(msg *discordgo.Message) string {
	if msg == nil || len(msg.Embeds) == 0 {
		return ""
	}

	for _, field := range msg.Embeds[0].Fields {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if name == "📍 location" || strings.Contains(name, "location") {
			return field.Value
		}
	}

	return ""
}
