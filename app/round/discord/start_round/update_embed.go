package startround

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (m *startRoundManager) UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (StartRoundOperationResult, error) {
	return m.operationWrapper(ctx, "UpdateRoundToScorecard", func(ctx context.Context) (StartRoundOperationResult, error) {
		// Multi-tenant: resolve channel ID from guild config if not provided
		resolvedChannelID := channelID
		if resolvedChannelID == "" && payload != nil && payload.GuildID != "" {
			cfg, err := m.guildConfigResolver.GetGuildConfigWithContext(ctx, string(payload.GuildID))
			if err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
			}
		}

		if resolvedChannelID == "" {
			err := errors.New("missing channel ID")
			m.logger.ErrorContext(ctx, "Missing channel ID for UpdateRoundToScorecard")
			return StartRoundOperationResult{Error: err}, nil
		}
		if messageID == "" {
			err := errors.New("missing message ID")
			m.logger.ErrorContext(ctx, "Missing message ID for UpdateRoundToScorecard")
			return StartRoundOperationResult{Error: err}, nil
		}

		// 🆕 Fetch existing message
		existingMsg, err := m.session.ChannelMessage(resolvedChannelID, messageID)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to fetch existing message for UpdateRoundToScorecard",
				attr.Error(err), attr.String("message_id", messageID))
			return StartRoundOperationResult{Error: fmt.Errorf("failed to fetch existing message: %w", err)}, nil
		}

		// 🆕 Pass current embed to the transformer
		existingEmbed := &discordgo.MessageEmbed{}
		if len(existingMsg.Embeds) > 0 {
			existingEmbed = existingMsg.Embeds[0]
		}

		transformResult, err := m.TransformRoundToScorecard(ctx, payload, existingEmbed)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to run TransformRoundToScorecard wrapper", attr.Error(err))
			return StartRoundOperationResult{}, fmt.Errorf("failed to run TransformRoundToScorecard wrapper: %w", err)
		}
		if transformResult.Error != nil {
			m.logger.ErrorContext(ctx, "TransformRoundToScorecard returned error in result", attr.Error(transformResult.Error))
			return StartRoundOperationResult{Error: fmt.Errorf("transformation failed: %w", transformResult.Error)}, nil
		}

		transformedData, ok := transformResult.Success.(struct {
			Embed      *discordgo.MessageEmbed
			Components []discordgo.MessageComponent
		})
		if !ok {
			err := errors.New("unexpected success type from TransformRoundToScorecard")
			m.logger.ErrorContext(ctx, "Unexpected type from TransformRoundToScorecard success", attr.Error(err))
			return StartRoundOperationResult{Error: err}, nil
		}

		_, err = m.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    resolvedChannelID,
			ID:         messageID,
			Embeds:     &[]*discordgo.MessageEmbed{transformedData.Embed},
			Components: &transformedData.Components,
		})
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to update round embed to scorecard",
				attr.Error(err), attr.String("message_id", messageID))
			return StartRoundOperationResult{Error: fmt.Errorf("failed to update round embed: %w", err)}, nil
		}

		m.logger.InfoContext(ctx, "Successfully updated round embed to scorecard",
			attr.String("message_id", messageID),
			attr.String("channel_id", resolvedChannelID))

		return StartRoundOperationResult{Success: true}, nil
	})
}
