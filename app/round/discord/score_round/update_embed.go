package scoreround

import (
	"context"
	"fmt"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "update_score_embed")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, string(userID))

	return srm.operationWrapper(ctx, "update_score_embed", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		if srm.session == nil {
			return ScoreRoundOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}

		message, err := srm.session.ChannelMessage(channelID, messageID)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to fetch message for score update",
				attr.Error(err),
				attr.String("channel_id", channelID),
				attr.String("message_id", messageID))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		if message.Embeds == nil {
			message.Embeds = []*discordgo.MessageEmbed{}
		}

		userFound := false

		for _, embed := range message.Embeds {
			if embed == nil {
				continue
			}

			user, err := srm.session.User(string(userID))
			if err != nil {
				continue
			}

			username := user.Username
			if member, err := srm.session.GuildMember(srm.config.Discord.GuildID, string(userID)); err == nil && member.Nick != "" {
				username = member.Nick
			}

			targetFieldName := fmt.Sprintf("🏌️ %s", username)
			for i, field := range embed.Fields {
				if field.Name == targetFieldName {
					if score != nil {
						embed.Fields[i].Value = fmt.Sprintf("Score: +%d", *score)
					} else {
						embed.Fields[i].Value = "Score: --"
					}
					userFound = true
					break
				}
			}
		}

		if !userFound {
			return ScoreRoundOperationResult{Success: "User not found in embed"}, nil
		}

		edit := &discordgo.MessageEdit{
			Channel: channelID,
			ID:      messageID,
		}

		if len(message.Embeds) > 0 {
			edit.SetEmbeds(message.Embeds)
		} else {
			edit.Embeds = &[]*discordgo.MessageEmbed{}
		}

		updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to update message with new score",
				attr.Error(err),
				attr.String("user_id", string(userID)))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		scoreValue := 0
		if score != nil {
			// Convert sharedtypes.Score to int for logging
			scoreValue = int(*score)
		}

		srm.logger.InfoContext(ctx, "Successfully updated user score in embed",
			attr.String("user_id", string(userID)),
			attr.Int("score", scoreValue))

		return ScoreRoundOperationResult{Success: updatedMsg}, nil
	})
}

func UpdateUserScoreInEmbed(ctx context.Context, session discord.Session, embed *discordgo.MessageEmbed, userID sharedtypes.DiscordID, score *sharedtypes.Score, guildID string) bool {
	if embed == nil {
		return false
	}

	for i := 0; i < len(embed.Fields); i++ {
		field := embed.Fields[i]

		user, err := session.User(string(userID))
		if err != nil {
			continue
		}

		var username string
		username = user.Username

		if member, err := session.GuildMember(guildID, string(userID)); err == nil && member.Nick != "" {
			username = member.Nick
		}

		targetFieldName := fmt.Sprintf("🏌️ %s", username)
		if field.Name == targetFieldName {
			if score != nil {
				embed.Fields[i].Value = fmt.Sprintf("Score: +%d", *score)
			} else {
				embed.Fields[i].Value = "Score: --"
			}
			return true
		}
	}
	return false
}
