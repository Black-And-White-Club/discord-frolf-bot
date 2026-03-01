package scoreround

import (
	"fmt"
	"strings"

	embedpagination "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/embed_pagination"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func messageHasPager(components []discordgo.MessageComponent) bool {
	for _, component := range components {
		row, ok := component.(discordgo.ActionsRow)
		if !ok {
			if rowPtr, ok := component.(*discordgo.ActionsRow); ok && rowPtr != nil {
				row = *rowPtr
			} else {
				continue
			}
		}
		for _, rowComponent := range row.Components {
			button, ok := rowComponent.(discordgo.Button)
			if !ok {
				if buttonPtr, ok := rowComponent.(*discordgo.Button); ok && buttonPtr != nil {
					button = *buttonPtr
				} else {
					continue
				}
			}
			if embedpagination.IsPagerCustomID(button.CustomID) {
				return true
			}
		}
	}
	return false
}

func storeLineSnapshotFromEmbed(messageID string, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	if embed == nil || messageID == "" {
		return
	}

	staticFields := make([]*discordgo.MessageEmbedField, 0, len(embed.Fields))
	participantLines := []string{}
	participantFieldName := "👥 Participants"

	for _, field := range embed.Fields {
		if field == nil {
			continue
		}
		if isParticipantField(field.Name, field.Value) {
			if field.Name != "" {
				participantFieldName = field.Name
			}
			participantLines = append(participantLines, embedpagination.ParticipantLinesFromFieldValue(field.Value)...)
			continue
		}
		staticFields = append(staticFields, field)
	}

	snapshot := embedpagination.NewLineSnapshot(
		messageID,
		embed,
		components,
		staticFields,
		participantFieldName,
		participantLines,
	)
	embedpagination.Set(snapshot)
}

func updateLineItemsScore(lines []string, userID sharedtypes.DiscordID, score *sharedtypes.Score) ([]string, bool) {
	if len(lines) == 0 {
		return lines, false
	}

	updated := make([]string, len(lines))
	copy(updated, lines)

	found := false
	for i, line := range updated {
		uid, _, tag, ok := parseParticipantLine(line)
		if !ok || uid != userID {
			continue
		}
		updated[i] = formatParticipantLine(uid, score, tag)
		found = true
	}

	return updated, found
}

func updateFieldItemsScore(fields []*discordgo.MessageEmbedField, userID sharedtypes.DiscordID, score *sharedtypes.Score) ([]*discordgo.MessageEmbedField, bool) {
	if len(fields) == 0 {
		return fields, false
	}

	updated := cloneEmbedFields(fields)
	found := false

	for i, field := range updated {
		if field == nil {
			continue
		}
		mentionID, mentionBlock, ok := parseMentionFromFinalizedValue(field.Value)
		if !ok || mentionID != string(userID) {
			continue
		}
		field.Value = formatFinalizedParticipantValue(field.Value, mentionBlock, score, false)
		updated[i] = field
		found = true
	}

	return updated, found
}

func updateFieldItemsFromParticipants(fields []*discordgo.MessageEmbedField, participants []roundtypes.Participant) []*discordgo.MessageEmbedField {
	updated := cloneEmbedFields(fields)
	if len(updated) == 0 {
		return updated
	}

	participantByUser := make(map[string]roundtypes.Participant, len(participants))
	for _, participant := range participants {
		participantByUser[string(participant.UserID)] = participant
	}

	for i, field := range updated {
		if field == nil {
			continue
		}
		mentionID, mentionBlock, ok := parseMentionFromFinalizedValue(field.Value)
		if !ok {
			continue
		}
		participant, found := participantByUser[mentionID]
		if !found {
			continue
		}
		field.Value = formatFinalizedParticipantValue(field.Value, mentionBlock, participant.Score, participant.IsDNF)
		updated[i] = field
	}

	return updated
}

func appendMissingParticipantLines(existing []string, participants []roundtypes.Participant) []string {
	if len(participants) == 0 {
		return existing
	}

	result := make([]string, len(existing))
	copy(result, existing)

	existingUsers := make(map[sharedtypes.DiscordID]struct{}, len(existing))
	for _, line := range existing {
		uid, _, _, ok := parseParticipantLine(line)
		if ok {
			existingUsers[uid] = struct{}{}
		}
	}

	for _, participant := range participants {
		if _, found := existingUsers[participant.UserID]; found {
			continue
		}
		result = append(result, formatParticipantLine(participant.UserID, participant.Score, participant.TagNumber))
	}

	return result
}

func parseMentionFromFinalizedValue(value string) (userID string, mentionBlock string, ok bool) {
	openIdx := strings.LastIndex(value, "(<@")
	closeIdx := strings.LastIndex(value, ")")
	if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx+3 {
		return "", "", false
	}

	mentionBlock = value[openIdx : closeIdx+1]
	mention := strings.TrimPrefix(strings.TrimSuffix(mentionBlock, ")"), "(")
	userID = strings.TrimSuffix(strings.TrimPrefix(mention, "<@"), ">")
	if userID == "" {
		return "", "", false
	}

	return userID, mentionBlock, true
}

func formatFinalizedParticipantValue(original, mentionBlock string, score *sharedtypes.Score, isDNF bool) string {
	prefix := strings.TrimSpace(strings.TrimSuffix(original, mentionBlock))
	pointsPart := ""
	if idx := strings.Index(prefix, "•"); idx != -1 {
		pointsPart = strings.TrimSpace(prefix[idx:])
	}

	if !isDNF && strings.Contains(strings.ToUpper(prefix), "SCORE: DNF") {
		isDNF = true
	}

	scorePart := fmt.Sprintf("%s %s", scorePrefix, scoreNoData)
	if isDNF {
		scorePart = fmt.Sprintf("%s DNF", scorePrefix)
	} else if score != nil {
		if *score == 0 {
			scorePart = fmt.Sprintf("%s Even", scorePrefix)
		} else {
			scorePart = fmt.Sprintf("%s %+d", scorePrefix, *score)
		}
	}

	if pointsPart != "" && !isDNF {
		scorePart = fmt.Sprintf("%s • %s", scorePart, strings.TrimPrefix(pointsPart, "• "))
	}

	return fmt.Sprintf("%s %s", scorePart, mentionBlock)
}

func cloneEmbedFields(fields []*discordgo.MessageEmbedField) []*discordgo.MessageEmbedField {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]*discordgo.MessageEmbedField, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		copyField := *field
		cloned = append(cloned, &copyField)
	}
	return cloned
}
