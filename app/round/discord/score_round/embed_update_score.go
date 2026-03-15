package scoreround

import (
	"context"
	"fmt"
	"strings"

	embedpagination "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/embed_pagination"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// parseParticipantLine parses a line in the embed to extract userID and optional tag.
func parseParticipantLine(line string) (userID sharedtypes.DiscordID, score *sharedtypes.Score, tag *sharedtypes.TagNumber, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "<@") || !strings.Contains(line, ">") {
		return "", nil, nil, false
	}
	end := strings.Index(line, ">")
	userID = sharedtypes.DiscordID(line[2:end])

	// Very simple parse for tag and score if formatted as: "<@ID> Tag: 5 — Score: +3"
	parts := strings.Split(line[end+1:], "—")
	if len(parts) > 0 {
		left := strings.TrimSpace(parts[0])
		if strings.HasPrefix(left, tagPrefix) {
			var t int
			// tagPrefix includes the colon ("Tag:"), so match that format.
			if _, err := fmt.Sscanf(left, tagPrefix+" %d", &t); err == nil {
				v := sharedtypes.TagNumber(t)
				tag = &v
			}
		}
	}
	if len(parts) > 1 {
		var s int
		right := strings.TrimSpace(parts[1])
		// scorePrefix includes the colon ("Score:"), accept optional sign.
		if _, err := fmt.Sscanf(right, scorePrefix+" %+d", &s); err != nil {
			if _, err2 := fmt.Sscanf(right, scorePrefix+" %d", &s); err2 == nil {
				v := sharedtypes.Score(s)
				score = &v
			}
		} else {
			v := sharedtypes.Score(s)
			score = &v
		}
	}
	return userID, score, tag, true
}

// UpdateScoreEmbed updates a single participant's score in an embed.
func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "update_score_embed", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message: %w", err)
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
		}

		if messageHasPager(message.Components) {
			if snapshot, found := embedpagination.Update(messageID, func(snapshot *embedpagination.Snapshot) bool {
				if snapshot == nil {
					return false
				}

				switch snapshot.Kind {
				case embedpagination.SnapshotKindFields:
					updatedFields, updated := updateFieldItemsScore(snapshot.FieldItems, userID, score)
					if !updated {
						return false
					}
					snapshot.FieldItems = updatedFields
					return true
				default:
					updatedLines, updated := updateLineItemsScore(snapshot.LineItems, userID, score)
					if !updated {
						return false
					}
					snapshot.LineItems = updatedLines
					return true
				}
			}); found {
				embed, components, _, _, renderErr := embedpagination.RenderPage(messageID, snapshot.CurrentPage)
				if renderErr != nil {
					return ScoreRoundOperationResult{Error: renderErr}, renderErr
				}

				edit := &discordgo.MessageEdit{
					Channel:    channelID,
					ID:         messageID,
					Embeds:     &[]*discordgo.MessageEmbed{embed},
					Components: &components,
				}

				updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
				if err != nil {
					return ScoreRoundOperationResult{Error: err}, err
				}

				return ScoreRoundOperationResult{Success: updatedMsg}, nil
			}
		}

		embed := message.Embeds[0]

		userFound := false
		for i := range embed.Fields {
			lines := strings.Split(embed.Fields[i].Value, "\n")
			for j, line := range lines {
				uid, _, tag, ok := parseParticipantLine(line)
				if !ok {
					continue
				}
				if uid == userID {
					lines[j] = formatParticipantLine(uid, score, tag)
					userFound = true
				}
			}
			embed.Fields[i].Value = strings.Join(lines, "\n")
		}

		if !userFound {
			return ScoreRoundOperationResult{Success: fmt.Sprintf("User %s not found in embed fields to update score", userID)}, nil
		}

		edit := &discordgo.MessageEdit{
			Channel: channelID,
			ID:      messageID,
		}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})

		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}

		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}

// UpdateScoreEmbedBulk updates multiple participants in an embed.
func (srm *scoreRoundManager) UpdateScoreEmbedBulk(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "update_score_embed_bulk", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "no embeds"}, nil
		}

		if messageHasPager(message.Components) {
			if snapshot, found := embedpagination.Update(messageID, func(snapshot *embedpagination.Snapshot) bool {
				if snapshot == nil {
					return false
				}

				switch snapshot.Kind {
				case embedpagination.SnapshotKindFields:
					snapshot.FieldItems = updateFieldItemsFromParticipants(snapshot.FieldItems, participants)
				default:
					snapshot.LineItems = participantsToEmbedLines(participants)
				}
				return true
			}); found {
				embed, components, _, _, renderErr := embedpagination.RenderPage(messageID, snapshot.CurrentPage)
				if renderErr != nil {
					return ScoreRoundOperationResult{Error: renderErr}, renderErr
				}

				edit := &discordgo.MessageEdit{
					Channel:    channelID,
					ID:         messageID,
					Embeds:     &[]*discordgo.MessageEmbed{embed},
					Components: &components,
				}

				updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
				if err != nil {
					return ScoreRoundOperationResult{Error: err}, err
				}

				return ScoreRoundOperationResult{Success: updatedMsg}, nil
			}
		}

		embed := message.Embeds[0]

		// Filter out all existing participant fields to "clean" the embed and avoid duplicates.
		var newFields []*discordgo.MessageEmbedField
		participantFieldIndex := -1

		for _, f := range embed.Fields {
			if isParticipantField(f.Name, f.Value) {
				// Stick to the first location we find for participants.
				if participantFieldIndex == -1 {
					participantFieldIndex = len(newFields)
					// Append a placeholder that we will overwrite shortly.
					newFields = append(newFields, f)
				}
				// Skip subsequent participant fields (consolidating them).
			} else {
				newFields = append(newFields, f)
			}
		}
		embed.Fields = newFields

		lines := participantsToEmbedLines(participants)
		newValue := strings.Join(lines, "\n")
		if len(lines) == 0 {
			newValue = placeholderNoParticipants
		}

		fieldName := "👥 Participants"
		if participantFieldIndex != -1 {
			embed.Fields[participantFieldIndex].Name = fieldName
			embed.Fields[participantFieldIndex].Value = newValue
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  fieldName,
				Value: newValue,
			})
		}

		edit := &discordgo.MessageEdit{
			Channel: channelID,
			ID:      messageID,
		}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})

		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}

		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}

// AddLateParticipantToScorecard appends new participants to an embed.
func (srm *scoreRoundManager) AddLateParticipantToScorecard(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "add_late_participant", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds to update"}, nil
		}

		if snapshot, found := embedpagination.Update(messageID, func(snapshot *embedpagination.Snapshot) bool {
			if snapshot == nil || snapshot.Kind != embedpagination.SnapshotKindLines {
				return false
			}
			snapshot.LineItems = appendMissingParticipantLines(snapshot.LineItems, participants)
			return true
		}); found {
			embed, components, _, _, renderErr := embedpagination.RenderPage(messageID, snapshot.CurrentPage)
			if renderErr != nil {
				return ScoreRoundOperationResult{Error: renderErr}, renderErr
			}

			edit := &discordgo.MessageEdit{
				Channel:    channelID,
				ID:         messageID,
				Embeds:     &[]*discordgo.MessageEmbed{embed},
				Components: &components,
			}

			updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
			if err != nil {
				return ScoreRoundOperationResult{Error: err}, err
			}

			return ScoreRoundOperationResult{Success: updatedMsg}, nil
		}

		embed := message.Embeds[0]

		// Consolidate all participant fields into one and append missing participants.
		var newFields []*discordgo.MessageEmbedField
		participantFieldIndex := -1
		var existingLines []string

		for _, f := range embed.Fields {
			if isParticipantField(f.Name, f.Value) {
				if participantFieldIndex == -1 {
					participantFieldIndex = len(newFields)
					newFields = append(newFields, f)
					existingLines = append(existingLines, embedpagination.ParticipantLinesFromFieldValue(f.Value)...)
				}
				// Skip subsequent participant fields (consolidating them).
			} else {
				newFields = append(newFields, f)
			}
		}
		embed.Fields = newFields

		updatedLines := appendMissingParticipantLines(existingLines, participants)
		newValue := strings.Join(updatedLines, "\n")
		if len(updatedLines) == 0 {
			newValue = placeholderNoParticipants
		}

		fieldName := "👥 Participants"
		if participantFieldIndex != -1 {
			embed.Fields[participantFieldIndex].Name = fieldName
			embed.Fields[participantFieldIndex].Value = newValue
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  fieldName,
				Value: newValue,
			})
		}

		storeLineSnapshotFromEmbed(messageID, embed, message.Components)

		edit := &discordgo.MessageEdit{
			Channel: channelID,
			ID:      messageID,
		}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})

		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}

		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}
