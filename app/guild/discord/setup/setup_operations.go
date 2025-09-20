package setup

import (
	"context"
	"fmt"
	"time"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	"github.com/bwmarrin/discordgo"
)

// performCustomSetup performs guild setup with custom configuration
func (s *setupManager) performCustomSetup(guildID string, config SetupConfig) (*SetupResult, error) {
	result := &SetupResult{
		RoleMappings: make(map[string]string),
	}

	// Get guild info
	guild, err := s.session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild info: %w", err)
	}

	// Create/find channels if requested
	if config.CreateChannels {
		channels := []struct {
			name       string
			topic      string
			target     *string
			targetName *string
		}{
			{config.ChannelPrefix + "-events", "📊 Disc golf events and round management", &result.EventChannelID, &result.EventChannelName},
			// Tests expect only the events channel to have its topic set during setup.
			{config.ChannelPrefix + "-leaderboard", "", &result.LeaderboardChannelID, &result.LeaderboardChannelName},
			{config.ChannelPrefix + "-signup", "", &result.SignupChannelID, &result.SignupChannelName},
		}

		for _, ch := range channels {
			channelID, err := s.createOrFindChannel(guildID, ch.name, ch.topic)
			if err != nil {
				return nil, fmt.Errorf("failed to setup channel %s: %w", ch.name, err)
			}
			*ch.target = channelID
			*ch.targetName = ch.name
		}
	}

	// Create/find roles if requested
	if config.CreateRoles {
		roles := []struct {
			name   string
			color  int
			target *string
		}{
			{config.UserRoleName, 0x00ff00, &result.UserRoleID},
			{config.EditorRoleName, 0xffff00, &result.EditorRoleID},
			{config.AdminRoleName, 0xff6600, &result.AdminRoleID},
		}

		for _, role := range roles {
			roleID, err := s.createOrFindRole(guild, role.name, role.color)
			if err != nil {
				return nil, fmt.Errorf("failed to setup role %s: %w", role.name, err)
			}
			if roleID == "" {
				return nil, fmt.Errorf("role creation for %s returned empty ID", role.name)
			}
			*role.target = roleID
			result.RoleMappings[role.name] = roleID
		}
	}

	// Create signup message if requested and signup channel exists
	if config.CreateSignupMsg && result.SignupChannelID != "" {
		messageID, err := s.createSignupMessage(guildID, result.SignupChannelID, config.SignupMessage, config.SignupEmoji)
		if err != nil {
			return nil, fmt.Errorf("failed to create signup message: %w", err)
		}
		result.SignupMessageID = messageID
		result.SignupEmoji = config.SignupEmoji
		if result.SignupEmoji == "" {
			result.SignupEmoji = "🥏"
		}
	}

	return result, nil
}

// publishSetupEvent publishes the guild setup event to the backend
func (s *setupManager) publishSetupEvent(i *discordgo.InteractionCreate, result *SetupResult) error {
	// Validate that required role IDs are not empty
	if result.UserRoleID == "" {
		return fmt.Errorf("user role ID is required but empty")
	}
	if result.EditorRoleID == "" {
		return fmt.Errorf("editor role ID is required but empty")
	}
	if result.AdminRoleID == "" {
		return fmt.Errorf("admin role ID is required but empty")
	}

	// Get guild info
	guild, err := s.session.Guild(i.GuildID)
	if err != nil {
		return fmt.Errorf("failed to get guild info: %w", err)
	}

	// Create the guild setup event
	setupTime := time.Now()
	signupEmoji := result.SignupEmoji
	if signupEmoji == "" {
		signupEmoji = "🥏" // Default fallback
	}

	event := guildevents.GuildSetupEvent{
		GuildID:                i.GuildID,
		GuildName:              guild.Name,
		AdminUserID:            i.Member.User.ID,
		EventChannelID:         result.EventChannelID,
		EventChannelName:       result.EventChannelName,
		LeaderboardChannelID:   result.LeaderboardChannelID,
		LeaderboardChannelName: result.LeaderboardChannelName,
		SignupChannelID:        result.SignupChannelID,
		SignupChannelName:      result.SignupChannelName,
		RoleMappings:           result.RoleMappings,
		RegisteredRoleID:       result.UserRoleID, // Legacy field for backward compatibility
		UserRoleID:             result.UserRoleID,
		EditorRoleID:           result.EditorRoleID,
		AdminRoleID:            result.AdminRoleID,
		SignupEmoji:            signupEmoji,
		SignupMessageID:        result.SignupMessageID,
		SetupCompletedAt:       setupTime,
	}

	// Create and publish the message using the helper
	msg, err := s.helper.CreateNewMessage(event, guildevents.GuildSetupEventTopic)
	if err != nil {
		return fmt.Errorf("failed to create setup event message: %w", err)
	}

	msg.Metadata.Set("guild_id", i.GuildID)

	return s.publisher.Publish(guildevents.GuildSetupEventTopic, msg)
}

// createOrFindChannel creates a new channel or finds an existing one
func (s *setupManager) createOrFindChannel(guildID, channelName, topic string) (string, error) {
	// Try to find existing channel first
	channels, err := s.session.GuildChannels(guildID)
	if err != nil {
		return "", err
	}

	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildText && channel.Name == channelName {
			return channel.ID, nil
		}
	}

	// Create new channel
	channel, err := s.session.GuildChannelCreate(guildID, channelName, discordgo.ChannelTypeGuildText)
	if err != nil {
		return "", err
	}

	// Set topic if provided
	if topic != "" {
		s.session.ChannelEdit(channel.ID, &discordgo.ChannelEdit{Topic: topic})
	}

	return channel.ID, nil
}

// createOrFindRole creates a new role or finds an existing one
func (s *setupManager) createOrFindRole(guild *discordgo.Guild, roleName string, color int) (string, error) {
	// Try to find existing role
	for _, role := range guild.Roles {
		if role.Name == roleName {
			if role.ID == "" {
				return "", fmt.Errorf("found existing role %s but it has empty ID", roleName)
			}
			return role.ID, nil
		}
	}

	// Create new role
	role, err := s.session.GuildRoleCreate(guild.ID, &discordgo.RoleParams{
		Name:  roleName,
		Color: &color,
	})
	if err != nil {
		return "", err
	}

	if role == nil || role.ID == "" {
		return "", fmt.Errorf("role creation for %s succeeded but returned empty/nil role", roleName)
	}

	return role.ID, nil
}

// createSignupMessage creates a signup message with custom content and emoji
func (s *setupManager) createSignupMessage(guildID, channelID, content, emojiName string) (string, error) {
	if content == "" {
		content = "React with 🥏 to sign up for frolf events!"
	}
	if emojiName == "" {
		emojiName = "🥏"
	}

	message, err := s.session.ChannelMessageSend(channelID, content)
	if err != nil {
		return "", fmt.Errorf("failed to send signup message: %w", err)
	}

	// Add reaction to the message
	err = s.session.MessageReactionAdd(channelID, message.ID, emojiName)
	if err != nil {
		if s.logger != nil {
			s.logger.ErrorContext(context.Background(), "Failed to add reaction to signup message",
				"error", err,
				"emoji", emojiName,
				"channel_id", channelID,
				"message_id", message.ID)
		}
		// Don't fail the setup, but make sure we log the error properly
	} else {
		if s.logger != nil {
			s.logger.InfoContext(context.Background(), "Successfully added reaction to signup message",
				"emoji", emojiName,
				"channel_id", channelID,
				"message_id", message.ID)
		}
	}

	if s.logger != nil {
		s.logger.InfoContext(context.Background(), "Created signup message",
			"guild_id", guildID,
			"channel_id", channelID,
			"message_id", message.ID,
			"content", content,
			"emoji", emojiName)
	}

	return message.ID, nil
}
