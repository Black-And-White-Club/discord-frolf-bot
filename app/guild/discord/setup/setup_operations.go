package setup

import (
	"fmt"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
			{config.ChannelPrefix + "-leaderboard", "🏆 Disc golf rankings and round results", &result.LeaderboardChannelID, &result.LeaderboardChannelName},
			{config.ChannelPrefix + "-signup", "✋ Sign up to participate in disc golf rounds", &result.SignupChannelID, &result.SignupChannelName},
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
			{config.PlayerRoleName, 0x00ff00, &result.RegisteredRoleID},
			{config.AdminRoleName, 0xff6600, &result.AdminRoleID},
		}

		for _, role := range roles {
			roleID, err := s.createOrFindRole(guild, role.name, role.color)
			if err != nil {
				return nil, fmt.Errorf("failed to setup role %s: %w", role.name, err)
			}
			*role.target = roleID
			result.RoleMappings[role.name] = roleID
		}
	}

	return result, nil
}

// publishSetupEvent publishes the guild setup event to the backend
func (s *setupManager) publishSetupEvent(i *discordgo.InteractionCreate, result *SetupResult) error {
	// Use the shared guild events to create a config creation request
	setupTime := time.Now()
	event := guildevents.GuildConfigRequestedPayload{
		GuildID:              sharedtypes.GuildID(i.GuildID),
		SignupChannelID:      result.SignupChannelID,
		SignupMessageID:      result.SignupMessageID,
		EventChannelID:       result.EventChannelID,
		LeaderboardChannelID: result.LeaderboardChannelID,
		UserRoleID:           result.RegisteredRoleID,
		EditorRoleID:         result.RegisteredRoleID, // Use same as user role for now
		AdminRoleID:          result.AdminRoleID,
		SignupEmoji:          "🥏", // Default emoji
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
	}

	// Create and publish the message using the helper
	msg, err := s.helper.CreateNewMessage(event, guildevents.GuildConfigCreationRequested)
	if err != nil {
		return fmt.Errorf("failed to create setup event message: %w", err)
	}

	msg.Metadata.Set("guild_id", i.GuildID)

	return s.publisher.Publish(guildevents.GuildConfigCreationRequested, msg)
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

	return role.ID, nil
}
