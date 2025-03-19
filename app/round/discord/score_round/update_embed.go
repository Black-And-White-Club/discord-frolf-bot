package scoreround

import (
	"context"
	"fmt"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

// UpdateScoreEmbed updates the score for a specific user in the scorecard embed
// It returns the updated message or an error if the update fails
func (srm *scoreRoundManager) UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID roundtypes.UserID, score *int) (*discordgo.Message, error) {
	// Ensure session is not nil to avoid panics
	if srm.session == nil {
		return nil, fmt.Errorf("session is nil")
	}

	// Fetch the original message
	message, err := srm.session.ChannelMessage(channelID, messageID)
	if err != nil {
		srm.logger.Error(ctx, "Failed to fetch message for score update",
			attr.Error(err),
			attr.String("channel_id", channelID),
			attr.String("message_id", messageID))
		return nil, err
	}

	// Ensure message.Embeds is not nil
	if message.Embeds == nil {
		message.Embeds = []*discordgo.MessageEmbed{}
	}

	// Flag to track if we found and updated the user's score
	userFound := false

	// Try to update the user's score in each embed
	for _, embed := range message.Embeds {
		if embed == nil {
			continue
		}

		// Get user information
		user, err := srm.session.User(string(userID))
		if err != nil {
			continue
		}

		// Get nickname if available
		username := user.Username
		if member, err := srm.session.GuildMember(srm.config.Discord.GuildID, string(userID)); err == nil && member.Nick != "" {
			username = member.Nick
		}

		// Check if user exists in the embed
		targetFieldName := fmt.Sprintf("🏌️ %s", username)
		for i, field := range embed.Fields {
			if field.Name == targetFieldName {
				// Found the user, update their score
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

	// If we didn't find the user, return the original message without editing
	if !userFound {
		return message, nil
	}

	// Create the edit structure
	edit := &discordgo.MessageEdit{
		Channel: channelID,
		ID:      messageID,
	}

	if len(message.Embeds) > 0 {
		edit.SetEmbeds(message.Embeds)
	} else {
		edit.Embeds = &[]*discordgo.MessageEmbed{}
	}

	// Send the update
	updatedMsg, err := srm.session.ChannelMessageEditComplex(edit)
	if err != nil {
		srm.logger.Error(ctx, "Failed to update message with new score",
			attr.Error(err),
			attr.String("user_id", string(userID)))
		return nil, err
	}

	// Ensure score is not nil before logging
	scoreValue := 0
	if score != nil {
		scoreValue = *score
	}

	srm.logger.Info(ctx, "Successfully updated user score in embed",
		attr.String("user_id", string(userID)),
		attr.Int("score", scoreValue))

	return updatedMsg, nil
}

// UpdateUserScoreInEmbed updates the score for a specific user in a message embed
// It uses the Discord session to fetch user details if needed
func UpdateUserScoreInEmbed(ctx context.Context, session discord.Session, embed *discordgo.MessageEmbed, userID string, score *int, guildID string) bool {
	if embed == nil {
		return false
	}

	// Skip header fields (typically the first few fields are for round info)
	// Start checking from participant fields
	for i := 0; i < len(embed.Fields); i++ {
		field := embed.Fields[i]

		// Try to match the field to the user
		// We need to check if this field belongs to the user we're updating
		// Fields have "🏌️ Username" format for the name
		user, err := session.User(userID)
		if err != nil {
			continue
		}

		// Get potential nicknames
		var username string
		username = user.Username

		if member, err := session.GuildMember(guildID, userID); err == nil && member.Nick != "" {
			username = member.Nick
		}

		// Check if this field belongs to our target user
		targetFieldName := fmt.Sprintf("🏌️ %s", username)
		if field.Name == targetFieldName {
			// Update the score
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
