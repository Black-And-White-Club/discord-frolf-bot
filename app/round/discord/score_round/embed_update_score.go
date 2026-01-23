package scoreround

import (
	"context"
	"fmt"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// parseParticipantLine parses a line in the embed to extract userID and optional tag
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
			// tagPrefix includes the colon ("Tag:"), so match that format
			if _, err := fmt.Sscanf(left, tagPrefix+" %d", &t); err == nil {
				v := sharedtypes.TagNumber(t)
				tag = &v
			}
		}
	}
	if len(parts) > 1 {
		var s int
		right := strings.TrimSpace(parts[1])
		// scorePrefix includes the colon ("Score:"), accept optional sign
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

// ============================
// Score Update Functions
// ============================

// UpdateScoreEmbed updates a single participant's score in an embed
func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "update_score_embed", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, fmt.Errorf("failed to fetch message: %w", err)
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds found to update"}, nil
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

// UpdateScoreEmbedBulk updates multiple participants in an embed
func (srm *scoreRoundManager) UpdateScoreEmbedBulk(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "update_score_embed_bulk", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "no embeds"}, nil
		}
		embed := message.Embeds[0]

		scoreByUser := make(map[sharedtypes.DiscordID]*sharedtypes.Score)
		tagByUser := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)
		for _, p := range participants {
			scoreByUser[p.UserID] = p.Score
			tagByUser[p.UserID] = p.TagNumber
		}

		updated := false
		for i := range embed.Fields {
			field := embed.Fields[i]
			if !isParticipantField(field.Name, field.Value) || strings.TrimSpace(field.Value) == "" || field.Value == placeholderNoParticipants {
				continue
			}

			lines := strings.Split(field.Value, "\n")
			for j, line := range lines {
				uid, oldScore, oldTag, ok := parseParticipantLine(line)
				if !ok {
					continue
				}
				newScore, scoreOK := scoreByUser[uid]
				newTag, tagOK := tagByUser[uid]
				if scoreOK || tagOK {
					if !scoreOK {
						newScore = oldScore
					}
					if !tagOK {
						newTag = oldTag
					}
					lines[j] = formatParticipantLine(uid, newScore, newTag)
					updated = true
				}
			}
			embed.Fields[i].Value = strings.Join(lines, "\n")
		}

		if !updated {
			return ScoreRoundOperationResult{Success: "no changes"}, nil
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

// AddLateParticipantToScorecard appends new participants to an embed
func (srm *scoreRoundManager) AddLateParticipantToScorecard(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error) {
	return srm.operationWrapper(ctx, "add_late_participant", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			return ScoreRoundOperationResult{Error: err}, err
		}
		if len(message.Embeds) == 0 {
			return ScoreRoundOperationResult{Success: "No embeds to update"}, nil
		}
		embed := message.Embeds[0]

		for _, p := range participants {
			added := false
			for i := range embed.Fields {
				if strings.TrimSpace(embed.Fields[i].Value) == "" || embed.Fields[i].Value == placeholderNoParticipants {
					embed.Fields[i].Value = strings.Join(participantsToEmbedLines([]roundtypes.Participant{p}), "\n")
					added = true
					break
				}
			}
			if !added {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:  "✅ Accepted",
					Value: strings.Join(participantsToEmbedLines([]roundtypes.Participant{p}), "\n"),
				})
			}
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
