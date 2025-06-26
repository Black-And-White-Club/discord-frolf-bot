package setup

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type SetupManager struct {
	session   *discordgo.Session
	publisher message.Publisher
	logger    *slog.Logger
}

func NewSetupManager(session *discordgo.Session, publisher message.Publisher, logger *slog.Logger) *SetupManager {
	return &SetupManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
	}
}

// HandleSetupCommand handles the /frolf-setup slash command
func (s *SetupManager) HandleSetupCommand(i *discordgo.InteractionCreate) error {
	// Check admin permissions
	if !s.hasAdminPermissions(i) {
		return s.respondError(i, "You need Administrator permissions to set up Frolf Bot")
	}

	// Try auto-setup first
	result, err := s.performAutoSetup(i.GuildID)
	if err != nil {
		s.logger.Error("Auto-setup failed", "guild_id", i.GuildID, "error", err)
		return s.respondError(i, fmt.Sprintf("Setup failed: %v", err))
	}

	// Publish setup event to backend
	if err := s.publishSetupEvent(i, result); err != nil {
		s.logger.Error("Failed to publish setup event", "guild_id", i.GuildID, "error", err)
		return s.respondError(i, "Setup completed but failed to save configuration")
	}

	// Respond with success
	return s.respondSuccess(i, result)
}

type SetupResult struct {
	EventChannelID         string
	EventChannelName       string
	LeaderboardChannelID   string
	LeaderboardChannelName string
	SignupChannelID        string
	SignupChannelName      string
	RegisteredRoleID       string
	AdminRoleID            string
	RoleMappings           map[string]string
}

func (s *SetupManager) performAutoSetup(guildID string) (*SetupResult, error) {
	result := &SetupResult{
		RoleMappings: make(map[string]string),
	}

	// Get guild info
	guild, err := s.session.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild info: %w", err)
	}

	// Create/find channels
	channels := []struct {
		name       string
		topic      string
		target     *string
		targetName *string
	}{
		{"frolf-events", "📊 Disc golf events and round management", &result.EventChannelID, &result.EventChannelName},
		{"frolf-leaderboard", "🏆 Disc golf rankings and tournament results", &result.LeaderboardChannelID, &result.LeaderboardChannelName},
		{"frolf-signup", "✋ Sign up to participate in disc golf rounds", &result.SignupChannelID, &result.SignupChannelName},
	}

	for _, ch := range channels {
		channelID, err := s.createOrFindChannel(guildID, ch.name, ch.topic)
		if err != nil {
			return nil, fmt.Errorf("failed to setup channel %s: %w", ch.name, err)
		}
		*ch.target = channelID
		*ch.targetName = ch.name
	}

	// Create/find roles
	roles := []struct {
		name   string
		color  int
		target *string
	}{
		{"Frolf Player", 0x00ff00, &result.RegisteredRoleID},
		{"Frolf Admin", 0xff6600, &result.AdminRoleID},
	}

	for _, role := range roles {
		roleID, err := s.createOrFindRole(guild, role.name, role.color)
		if err != nil {
			return nil, fmt.Errorf("failed to setup role %s: %w", role.name, err)
		}
		*role.target = roleID
		result.RoleMappings[role.name] = roleID
	}

	return result, nil
}

func (s *SetupManager) publishSetupEvent(i *discordgo.InteractionCreate, result *SetupResult) error {
	guild, err := s.session.Guild(i.GuildID)
	if err != nil {
		return fmt.Errorf("failed to get guild info: %w", err)
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
		RegisteredRoleID:       result.RegisteredRoleID,
		AdminRoleID:            result.AdminRoleID,
		SetupCompletedAt:       time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal setup event: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("guild_id", i.GuildID)

	return s.publisher.Publish("guild.setup", msg)
}

// Helper methods
func (s *SetupManager) hasAdminPermissions(i *discordgo.InteractionCreate) bool {
	return i.Member.Permissions&discordgo.PermissionAdministrator != 0
}

func (s *SetupManager) createOrFindChannel(guildID, channelName, topic string) (string, error) {
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

func (s *SetupManager) createOrFindRole(guild *discordgo.Guild, roleName string, color int) (string, error) {
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

func (s *SetupManager) respondSuccess(i *discordgo.InteractionCreate, result *SetupResult) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🥏 Frolf Bot Setup Complete!",
		Description: "Your server is ready for disc golf! Here's what I've set up:",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📊 Events Channel", Value: fmt.Sprintf("<#%s>", result.EventChannelID), Inline: true},
			{Name: "🏆 Leaderboard Channel", Value: fmt.Sprintf("<#%s>", result.LeaderboardChannelID), Inline: true},
			{Name: "✋ Signup Channel", Value: fmt.Sprintf("<#%s>", result.SignupChannelID), Inline: true},
			{Name: "👥 Player Role", Value: fmt.Sprintf("<@&%s>", result.RegisteredRoleID), Inline: true},
			{Name: "⚡ Admin Role", Value: fmt.Sprintf("<@&%s>", result.AdminRoleID), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /create-round to get started!",
		},
	}

	return s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (s *SetupManager) respondError(i *discordgo.InteractionCreate, errMsg string) error {
	return s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", errMsg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
