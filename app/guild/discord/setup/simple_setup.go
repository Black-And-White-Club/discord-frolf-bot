package setup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// SetupManager interface defines the contract for guild setup management
type SetupManager interface {
	HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error
}

type setupManager struct {
	session          *discordgo.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) error) error
}

// NewSetupManager creates a new SetupManager instance
func NewSetupManager(
	session *discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (SetupManager, error) {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating Guild SetupManager")
	}

	return &setupManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) error) error {
			return wrapSetupOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}, nil
}

// HandleSetupCommand handles the /frolf-setup slash command
func (s *setupManager) HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.operationWrapper(ctx, "handle_setup_command", func(ctx context.Context) error {
		// Check admin permissions
		if !s.hasAdminPermissions(i) {
			return s.respondError(i, "You need Administrator permissions to set up Frolf Bot")
		}

		// Try auto-setup first
		result, err := s.performAutoSetup(i.GuildID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Auto-setup failed", "guild_id", i.GuildID, "error", err)
			return s.respondError(i, fmt.Sprintf("Setup failed: %v", err))
		}

		// Publish setup event to backend
		if err := s.publishSetupEvent(ctx, i, result); err != nil {
			s.logger.ErrorContext(ctx, "Failed to publish setup event", "guild_id", i.GuildID, "error", err)
			return s.respondError(i, "Setup completed but failed to save configuration")
		}

		// Respond with success
		return s.respondSuccess(i, result)
	})
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

func (s *setupManager) performAutoSetup(guildID string) (*SetupResult, error) {
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

func (s *setupManager) publishSetupEvent(ctx context.Context, i *discordgo.InteractionCreate, result *SetupResult) error {
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

	// Create and publish the message using the helper
	msg, err := s.helper.CreateNewMessage(event, guildevents.GuildSetupEventTopic)
	if err != nil {
		return fmt.Errorf("failed to create setup event message: %w", err)
	}

	msg.Metadata.Set("guild_id", i.GuildID)

	return s.publisher.Publish(guildevents.GuildSetupEventTopic, msg)
}

// Helper methods
func (s *setupManager) hasAdminPermissions(i *discordgo.InteractionCreate) bool {
	return i.Member.Permissions&discordgo.PermissionAdministrator != 0
}

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

func (s *setupManager) respondSuccess(i *discordgo.InteractionCreate, result *SetupResult) error {
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

func (s *setupManager) respondError(i *discordgo.InteractionCreate, errMsg string) error {
	return s.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", errMsg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// wrapSetupOperation wraps setup operations with observability and error handling
func wrapSetupOperation(
	ctx context.Context,
	opName string,
	fn func(ctx context.Context) error,
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) error {
	// Start span for tracing
	ctx, span := tracer.Start(ctx, fmt.Sprintf("guild.setup.%s", opName))
	defer span.End()

	start := time.Now()

	// Execute the operation
	err := fn(ctx)

	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		if logger != nil {
			logger.ErrorContext(ctx, "Guild setup operation failed",
				"operation", opName,
				"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()),
				"error", err)
		}
	} else {
		if logger != nil {
			logger.InfoContext(ctx, "Guild setup operation completed",
				"operation", opName,
				"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()))
		}
	}

	return err
}
