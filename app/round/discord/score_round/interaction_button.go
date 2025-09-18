package scoreround

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// HandleScoreButton opens the score submission or bulk override modal.
func (srm *scoreRoundManager) HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		return ScoreRoundOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	srm.logger.InfoContext(ctx, "Handling score button interaction", attr.UserID(sharedtypes.DiscordID(userID)))

	return srm.operationWrapper(ctx, "handle_score_button", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			err := fmt.Errorf("invalid custom ID for score button: %s", customID)
			srm.logger.ErrorContext(ctx, "Invalid CustomID for score button", attr.Error(err))
			return ScoreRoundOperationResult{Error: err}, nil
		}
		roundID := parts[1]

		resolvedChannelID := i.ChannelID
		if i.GuildID != "" {
			if cfg, err := srm.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID); err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
			}
		}

		isOverride := strings.HasPrefix(customID, bulkOverrideButtonPrefix)
		if isOverride && !canOverrideFinalized(i.Member, srm.config) {
			_ = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "❌ Round is finalized. Only admins/editors can update scores.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return ScoreRoundOperationResult{Success: "Finalized round permission denied"}, nil
		}

		var modal *discordgo.InteractionResponse
		if isOverride {
			var prefill string
			if i.Message != nil && len(i.Message.Embeds) > 0 {
				parsed := parseFinalizedEmbedParticipants(i.Message.Embeds[0])
				prefill = strings.Join(buildPrefillLines(parsed), "\n")
			}
			if prefill == "" {
				prefill = "@UserA=-3\n@UserB=0\n123456789012345678=+5"
			}
			modal = buildBulkOverrideModal(roundID, userID, prefill)
		} else {
			modal = buildSingleScoreModal(roundID, userID)
		}

		if err := srm.session.InteractionRespond(i.Interaction, modal); err != nil {
			srm.logger.ErrorContext(ctx, "Failed to open score modal", attr.Error(err), attr.String("round_id", roundID), attr.String("user_id", userID), attr.String("channel_id", resolvedChannelID))
			return ScoreRoundOperationResult{Error: err}, nil
		}
		return ScoreRoundOperationResult{Success: "Score modal opened successfully"}, nil
	})
}

func buildBulkOverrideModal(roundID, userID, prefill string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{Title: "Score Override(s)", CustomID: bulkOverrideModalCustomID(roundID, userID), Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "bulk_scores_input", Label: "Score Overrides", Placeholder: "<@user|id|name|prefix>=score (0 allowed, -- keep, suffix ! force, # comment)", Style: discordgo.TextInputParagraph, Required: false, Value: prefill}}}}}}
}

func buildSingleScoreModal(roundID, userID string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &discordgo.InteractionResponseData{Title: "Submit Your Score", CustomID: singleScoreModalCustomID(roundID, userID), Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: "score_input", Label: "Enter your score (e.g., -3, 0, +5)", Style: discordgo.TextInputShort, Required: true, Placeholder: "Enter your disc golf score"}}}}}}
}

func bulkOverrideModalCustomID(roundID, userID string) string {
	return submitBulkOverridePrefix + roundID + "|" + userID
}

func singleScoreModalCustomID(roundID, userID string) string {
	return submitSingleModalPrefix + roundID + "|" + userID
}
