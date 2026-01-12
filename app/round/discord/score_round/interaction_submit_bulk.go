package scoreround

import (
	"context"
	"fmt"
	"strings"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// handleBulkScoreSubmission processes a bulk override modal submission.
func (srm *scoreRoundManager) handleBulkScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate, roundIDStr, userIDFromModal string) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_bulk")
	roundUUID, err := uuid.Parse(roundIDStr)
	if err != nil {
		return ScoreRoundOperationResult{Error: fmt.Errorf("invalid round ID in bulk override")}, nil
	}

	data := i.ModalSubmitData()
	var bulkValue string
	for _, comp := range data.Components {
		switch rowTyped := comp.(type) {
		case *discordgo.ActionsRow:
			for _, inner := range rowTyped.Components {
				if ti, ok := inner.(*discordgo.TextInput); ok && ti.CustomID == "bulk_scores_input" {
					bulkValue = ti.Value
				}
			}
		case discordgo.ActionsRow:
			for _, inner := range rowTyped.Components {
				if ti, ok := inner.(discordgo.TextInput); ok && ti.CustomID == "bulk_scores_input" {
					bulkValue = ti.Value
				}
			}
		}
	}

	var activeEmbed *discordgo.MessageEmbed
	if i.Message != nil && len(i.Message.Embeds) > 0 {
		activeEmbed = i.Message.Embeds[0]
	}
	originalParsed := parseFinalizedEmbedParticipants(activeEmbed)
	originalScores := map[string]int{}
	for uid, sc := range originalParsed {
		if sc != nil {
			originalScores[uid] = *sc
		}
	}

	nameToID := map[string]string{}
	participantIDs := map[string]struct{}{}
	if activeEmbed != nil {
		for _, f := range activeEmbed.Fields {
			if f == nil || f.Value == "" || !strings.Contains(f.Value, "(<@") {
				continue
			}
			openIdx := strings.LastIndex(f.Value, "(<@")
			closeIdx := strings.LastIndex(f.Value, ")")
			if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx+3 {
				continue
			}
			mention := f.Value[openIdx+1 : closeIdx]
			uid := strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(mention, "<@"), ">"), "!")
			participantIDs[uid] = struct{}{}
		}
	}
	if i.Member != nil && i.Member.User != nil {
		for _, sn := range []string{i.Member.Nick, i.Member.User.GlobalName, i.Member.User.Username} {
			if strings.TrimSpace(sn) == "" {
				continue
			}
			ln := strings.ToLower(strings.TrimSpace(sn))
			if _, exists := nameToID[ln]; !exists {
				nameToID[ln] = i.Member.User.ID
			}
		}
	}
	for uid := range participantIDs {
		member, mErr := srm.session.GuildMember(i.GuildID, uid)
		if mErr != nil || member == nil || member.User == nil {
			continue
		}
		display := member.Nick
		if strings.TrimSpace(display) == "" {
			display = member.User.GlobalName
		}
		if strings.TrimSpace(display) == "" {
			display = member.User.Username
		}
		if strings.TrimSpace(display) == "" {
			continue
		}
		ln := strings.ToLower(strings.TrimSpace(display))
		nameToID[ln] = uid
	}
	idx := buildNameIndex(nameToID)

	updates, unchanged, skipped, unresolved, diagnostics := parseBulkOverrides(bulkValue, originalScores, idx)
	var resolvedMappings []string
	for _, d := range diagnostics {
		if strings.Contains(d.reason, "resolved") {
			resolvedMappings = append(resolvedMappings, d.line)
		}
	}
	summary := summarizeBulk(updates, unchanged, skipped, unresolved, diagnostics, resolvedMappings)

	if len(updates) > 0 {
		bulkUpdates := make([]scoreevents.ScoreUpdateRequestPayload, 0, len(updates))
		for _, u := range updates {
			bulkUpdates = append(bulkUpdates, scoreevents.ScoreUpdateRequestPayload{
				GuildID: sharedtypes.GuildID(i.GuildID),
				RoundID: sharedtypes.RoundID(roundUUID),
				UserID:  sharedtypes.DiscordID(u.UserID),
				Score:   sharedtypes.Score(u.Score),
			})
		}
		bulkPayload := scoreevents.ScoreBulkUpdateRequestPayload{GuildID: sharedtypes.GuildID(i.GuildID), RoundID: sharedtypes.RoundID(roundUUID), Updates: bulkUpdates}
		msg := message.NewMessage(watermill.NewUUID(), nil)
		if msg.Metadata == nil {
			msg.Metadata = message.Metadata{}
		}
		msg.Metadata.Set("guild_id", i.GuildID)
		msg.Metadata.Set("override", "true")
		msg.Metadata.Set("override_mode", "bulk")
		msg.Metadata.Set("topic", scoreevents.ScoreBulkUpdateRequest)
		msg.Metadata.Set("user_id", userIDFromModal)
		if i.Message != nil {
			msg.Metadata.Set("message_id", i.Message.ID)
			msg.Metadata.Set("message_id", i.Message.ID)
		}
		if i.ChannelID != "" {
			msg.Metadata.Set("channel_id", i.ChannelID)
		}
		resultMsg, errCreate := srm.helper.CreateResultMessage(msg, bulkPayload, scoreevents.ScoreBulkUpdateRequest)
		if errCreate == nil {
			if errPub := srm.publisher.Publish(scoreevents.ScoreBulkUpdateRequest, resultMsg); errPub != nil {
				summary = "Bulk override failed to publish: " + errPub.Error()
			}
		} else {
			summary = "Bulk override failed to create message: " + errCreate.Error()
		}
	}

	respErr := srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: summary, Flags: discordgo.MessageFlagsEphemeral}})
	if respErr != nil {
		srm.logger.ErrorContext(ctx, "Failed to respond to bulk override modal", attr.Error(respErr))
		return ScoreRoundOperationResult{Error: respErr}, nil
	}
	return ScoreRoundOperationResult{Success: fmt.Sprintf("bulk override processed: %d", len(updates))}, nil
}
