package scoreround

import (
	"context"
	"fmt"
	"strings"

	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// UpdateScoreEmbed updates participant score in a scorecard embed (supports finalized & non-finalized formats).
func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_score_embed")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(userID))
	return srm.operationWrapper(ctx, "update_score_embed", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		if srm.session == nil {
			return ScoreRoundOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}

		resolvedChannelID := channelID
		if resolvedChannelID == "" {
			if guildID, _ := ctx.Value("guild_id").(string); guildID != "" {
				if cfg, err := srm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && cfg != nil && cfg.EventChannelID != "" {
					resolvedChannelID = cfg.EventChannelID
				}
			}
		}

		message, err := srm.session.ChannelMessage(resolvedChannelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message for score update: %w", err)
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
		}
		embed := message.Embeds[0]

		finalizedUpdated := false
		userFound := false
		for i, f := range embed.Fields {
			val := f.Value
			if strings.TrimSpace(val) == "" || !strings.Contains(val, "(<@") || !strings.Contains(val, scorePrefix) {
				continue
			}
			mentionA := "(<@" + string(userID) + ")"
			mentionB := "(<@!" + string(userID) + ")"
			if !(strings.Contains(val, mentionA) || strings.Contains(val, mentionB)) {
				continue
			}
			openIdx := strings.LastIndex(val, "(<@")
			closeIdx := strings.LastIndex(val, ")")
			if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
				continue
			}
			mentionPart := val[openIdx : closeIdx+1]
			var display string
			if score == nil {
				display = scoreNoData
			} else if *score == 0 {
				display = "Even"
			} else {
				display = fmt.Sprintf("%+d", *score)
			}
			newVal := fmt.Sprintf("%s %s %s", scorePrefix, display, mentionPart)
			if newVal != val {
				embed.Fields[i].Value = newVal
			}
			finalizedUpdated = true
			userFound = true
			break
		}
		if finalizedUpdated {
			edit := &discordgo.MessageEdit{Channel: resolvedChannelID, ID: messageID}
			edit.SetEmbeds([]*discordgo.MessageEmbed{embed})
			updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
			if err != nil {
				return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to edit message for score update: %w", err)
			}
			return ScoreRoundOperationResult{Success: updatedMsg}, nil
		}

		participantFields := []*discordgo.MessageEmbedField{}
		fieldNameMap := map[string]int{}
		for i, field := range embed.Fields {
			ln := strings.ToLower(field.Name)
			if strings.Contains(ln, "accepted") || strings.Contains(ln, "tentative") || strings.Contains(ln, "declined") || strings.Contains(ln, "participants") || strings.Contains(ln, "✅") || strings.Contains(ln, "❓") || strings.Contains(ln, "❌") {
				participantFields = append(participantFields, field)
				fieldNameMap[field.Name] = i
			}
		}
		if len(participantFields) == 0 {
			for i, field := range embed.Fields {
				if strings.Contains(field.Value, "<@") {
					participantFields = append(participantFields, field)
					fieldNameMap[field.Name] = i
				}
			}
		}
		if len(participantFields) == 0 {
			return ScoreRoundOperationResult{Success: "No participant fields found"}, nil
		}

		updatedFieldValues := map[string]string{}
		for _, field := range participantFields {
			if strings.TrimSpace(field.Value) == "" || field.Value == placeholderNoParticipants {
				updatedFieldValues[field.Name] = field.Value
				continue
			}
			originalLines := strings.Split(field.Value, "\n")
			newLines := []string{}
			for _, line := range originalLines {
				parsedUserID, parsedScore, parsedTag, ok := srm.parseParticipantLine(ctx, line)
				if !ok {
					newLines = append(newLines, line)
					continue
				}
				if parsedUserID == userID {
					userFound = true
					scoreDisplay := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
					if score != nil {
						scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *score)
					}
					tagDisplay := ""
					if parsedTag != nil && *parsedTag > 0 {
						tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *parsedTag)
					}
					newLines = append(newLines, fmt.Sprintf("<@%s>%s%s", userID, tagDisplay, scoreDisplay))
				} else {
					scoreDisplay := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
					if parsedScore != nil {
						scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *parsedScore)
					}
					tagDisplay := ""
					if parsedTag != nil && *parsedTag > 0 {
						tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *parsedTag)
					}
					newLines = append(newLines, fmt.Sprintf("<@%s>%s%s", parsedUserID, tagDisplay, scoreDisplay))
				}
			}
			if len(newLines) == 0 {
				updatedFieldValues[field.Name] = field.Value
			} else {
				updatedFieldValues[field.Name] = strings.Join(newLines, "\n")
			}
		}

		if !userFound {
			return ScoreRoundOperationResult{Success: fmt.Sprintf("User %s not found in embed fields to update score", userID)}, nil
		}
		for i, field := range embed.Fields {
			if newVal, ok := updatedFieldValues[field.Name]; ok {
				embed.Fields[i].Value = newVal
			}
		}
		edit := &discordgo.MessageEdit{Channel: resolvedChannelID, ID: messageID}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})
		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to edit message for score update: %w", err)
		}
		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}

// UpdateScoreEmbedBulk updates multiple participant scores in a single embed edit.
func (srm *scoreRoundManager) UpdateScoreEmbedBulk(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_score_embed_bulk")
	return srm.operationWrapper(ctx, "update_score_embed_bulk", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		if srm.session == nil {
			return ScoreRoundOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}

		scoreByUser := make(map[sharedtypes.DiscordID]*sharedtypes.Score, len(participants))
		tagByUser := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber, len(participants))
		for _, participant := range participants {
			id := participant.UserID
			if participant.Score != nil {
				s := *participant.Score
				scoreByUser[id] = &s
			} else {
				scoreByUser[id] = nil
			}
			if participant.TagNumber != nil {
				tn := *participant.TagNumber
				tagByUser[id] = &tn
			}
		}

		resolvedChannelID := channelID
		if resolvedChannelID == "" {
			if guildID, _ := ctx.Value("guild_id").(string); guildID != "" {
				if cfg, err := srm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID); err == nil && cfg != nil && cfg.EventChannelID != "" {
					resolvedChannelID = cfg.EventChannelID
				}
			}
		}

		message, err := srm.session.ChannelMessage(resolvedChannelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message for bulk score update: %w", err)
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
		}
		embed := message.Embeds[0]

		updatedAny := false
		for i, f := range embed.Fields {
			val := f.Value
			if strings.TrimSpace(val) == "" || !strings.Contains(val, "(<@") || !strings.Contains(val, scorePrefix) {
				continue
			}
			openIdx := strings.LastIndex(val, "<@")
			closeIdx := strings.LastIndex(val, ">")
			if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
				continue
			}
			userSegment := strings.TrimPrefix(val[openIdx+2:closeIdx], "!")
			userID := sharedtypes.DiscordID(userSegment)
			score, ok := scoreByUser[userID]
			if !ok {
				continue
			}
			mentionStart := strings.LastIndex(val, "(<@")
			mentionEnd := strings.LastIndex(val, ")")
			if mentionStart == -1 || mentionEnd == -1 || mentionEnd <= mentionStart {
				continue
			}
			mentionPart := val[mentionStart : mentionEnd+1]
			var display string
			if score == nil {
				display = scoreNoData
			} else if *score == 0 {
				display = "Even"
			} else {
				display = fmt.Sprintf("%+d", *score)
			}
			newVal := fmt.Sprintf("%s %s %s", scorePrefix, display, mentionPart)
			if newVal != val {
				embed.Fields[i].Value = newVal
				updatedAny = true
			}
		}

		participantFields := []*discordgo.MessageEmbedField{}
		fieldNameMap := map[string]int{}
		for i, field := range embed.Fields {
			ln := strings.ToLower(field.Name)
			if strings.Contains(ln, "accepted") || strings.Contains(ln, "tentative") || strings.Contains(ln, "declined") || strings.Contains(ln, "participants") || strings.Contains(ln, "✅") || strings.Contains(ln, "❓") || strings.Contains(ln, "❌") {
				participantFields = append(participantFields, field)
				fieldNameMap[field.Name] = i
			}
		}
		if len(participantFields) == 0 {
			for i, field := range embed.Fields {
				if strings.Contains(field.Value, "<@") {
					participantFields = append(participantFields, field)
					fieldNameMap[field.Name] = i
				}
			}
		}

		if len(participantFields) > 0 {
			updatedFieldValues := map[string]string{}
			for _, field := range participantFields {
				if strings.TrimSpace(field.Value) == "" || field.Value == placeholderNoParticipants {
					updatedFieldValues[field.Name] = field.Value
					continue
				}
				originalLines := strings.Split(field.Value, "\n")
				newLines := make([]string, 0, len(originalLines))
				for _, line := range originalLines {
					parsedUserID, parsedScore, parsedTag, ok := srm.parseParticipantLine(ctx, line)
					if !ok {
						newLines = append(newLines, line)
						continue
					}
					newScore, ok := scoreByUser[parsedUserID]
					if !ok {
						newLines = append(newLines, line)
						continue
					}
					tagDisplay := ""
					if tag, ok := tagByUser[parsedUserID]; ok && tag != nil && *tag > 0 {
						tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *tag)
					} else if parsedTag != nil && *parsedTag > 0 {
						tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *parsedTag)
					}
					scoreDisplay := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
					if newScore != nil {
						scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *newScore)
					} else if parsedScore != nil {
						scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *parsedScore)
					}
					newLines = append(newLines, fmt.Sprintf("<@%s>%s%s", parsedUserID, tagDisplay, scoreDisplay))
					updatedAny = true
				}
				if len(newLines) == 0 {
					updatedFieldValues[field.Name] = field.Value
				} else {
					updatedFieldValues[field.Name] = strings.Join(newLines, "\n")
				}
			}
			for i, field := range embed.Fields {
				if newVal, ok := updatedFieldValues[field.Name]; ok {
					embed.Fields[i].Value = newVal
				}
			}
		}

		if !updatedAny {
			return ScoreRoundOperationResult{Success: "No score updates applied"}, nil
		}

		edit := &discordgo.MessageEdit{Channel: resolvedChannelID, ID: messageID}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})
		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to edit message for bulk score update: %w", err)
		}
		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}

// AddLateParticipantToScorecard adds late participants.
func (srm *scoreRoundManager) AddLateParticipantToScorecard(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "add_late_participant")
	return srm.operationWrapper(ctx, "add_late_participant", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		if srm.session == nil {
			return ScoreRoundOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message: %w", err)
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
		}
		embed := message.Embeds[0]

		acceptedFieldIndex := -1
		tentativeFieldIndex := -1
		for i, field := range embed.Fields {
			ln := strings.ToLower(field.Name)
			if field.Name == "✅ Accepted" || strings.Contains(ln, "accepted") {
				acceptedFieldIndex = i
			}
			if field.Name == "🤔 Tentative" || strings.Contains(ln, "tentative") {
				tentativeFieldIndex = i
			}
		}
		if acceptedFieldIndex == -1 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "✅ Accepted", Value: placeholderNoParticipants})
			acceptedFieldIndex = len(embed.Fields) - 1
		}
		if tentativeFieldIndex == -1 {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "🤔 Tentative", Value: placeholderNoParticipants})
			tentativeFieldIndex = len(embed.Fields) - 1
		}

		added := 0
		for _, participant := range participants {
			var targetFieldIndex int
			switch participant.Response {
			case roundtypes.ResponseAccept:
				targetFieldIndex = acceptedFieldIndex
			case roundtypes.ResponseTentative:
				targetFieldIndex = tentativeFieldIndex
			default:
				continue
			}
			targetField := embed.Fields[targetFieldIndex]
			existingLines := []string{}
			if strings.TrimSpace(targetField.Value) != "" && targetField.Value != placeholderNoParticipants {
				existingLines = strings.Split(targetField.Value, "\n")
			}
			participantIDStr := string(participant.UserID)
			exists := false
			for _, line := range existingLines {
				if strings.Contains(line, participantIDStr) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			tagDisplay := ""
			if participant.TagNumber != nil && *participant.TagNumber > 0 {
				tagDisplay = fmt.Sprintf(" %s %d", tagPrefix, *participant.TagNumber)
			}
			scoreDisplay := fmt.Sprintf(" — %s %s", scorePrefix, scoreNoData)
			if participant.Score != nil {
				scoreDisplay = fmt.Sprintf(" — %s %+d", scorePrefix, *participant.Score)
			}
			newLine := fmt.Sprintf("<@%s>%s%s", participantIDStr, tagDisplay, scoreDisplay)
			existingLines = append(existingLines, newLine)
			embed.Fields[targetFieldIndex].Value = strings.Join(existingLines, "\n")
			added++
		}
		edit := &discordgo.MessageEdit{Channel: channelID, ID: messageID}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})
		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to edit message: %w", err)
		}
		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}
