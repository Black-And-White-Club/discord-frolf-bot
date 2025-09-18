// interactions/registry.go
package interactions

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

type PermissionLevel int

const (
	NoPermissionRequired PermissionLevel = iota
	PlayerRequired
	EditorRequired
	AdminRequired
)

type HandlerConfig struct {
	Handler            func(ctx context.Context, i *discordgo.InteractionCreate)
	RequiredPermission PermissionLevel
	RequiresSetup      bool // Whether the guild must be configured to use this command
}

type Registry struct {
	handlers            map[string]HandlerConfig
	guildConfigResolver guildconfig.GuildConfigResolver
	logger              *slog.Logger
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]HandlerConfig),
	}
}

// SetGuildConfigResolver sets the resolver for permission checking
func (r *Registry) SetGuildConfigResolver(resolver guildconfig.GuildConfigResolver) {
	r.guildConfigResolver = resolver
}

// SetLogger sets the logger for the registry
func (r *Registry) SetLogger(logger *slog.Logger) {
	r.logger = logger
}

func (r *Registry) RegisterHandler(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate)) {
	r.handlers[id] = HandlerConfig{
		Handler:            handler,
		RequiredPermission: NoPermissionRequired,
		RequiresSetup:      false,
	}
}

// RegisterHandlerWithPermissions registers a handler with permission requirements
func (r *Registry) RegisterHandlerWithPermissions(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate), permission PermissionLevel, requiresSetup bool) {
	r.handlers[id] = HandlerConfig{
		Handler:            handler,
		RequiredPermission: permission,
		RequiresSetup:      requiresSetup,
	}
}

func (r *Registry) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	var id string

	if r.logger != nil {
		r.logger.Info("🎯 Interaction received",
			slog.String("type", i.Type.String()),
			slog.String("guild_id", i.GuildID),
			slog.String("user_id", func() string {
				if i.Member != nil && i.Member.User != nil {
					return i.Member.User.ID
				} else if i.User != nil {
					return i.User.ID
				}
				return ""
			}()),
		)
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		id = i.ApplicationCommandData().Name
		if r.logger != nil {
			r.logger.Info("📋 Application command", slog.String("command", id))
		}
	case discordgo.InteractionMessageComponent:
		id = i.MessageComponentData().CustomID
		if r.logger != nil {
			r.logger.Info("🔘 Message component", slog.String("custom_id", id))
		}
	case discordgo.InteractionModalSubmit:
		// Extract modal submission data
		modalData := i.ModalSubmitData()
		if modalData.CustomID == "" {
			if r.logger != nil {
				r.logger.Error("❌ Modal submission data is invalid: CustomID is empty")
			}
			return
		}
		id = modalData.CustomID
		if r.logger != nil {
			r.logger.Info("📝 Modal submit", slog.String("custom_id", id))
		}
	}

	// Find the handler
	var config HandlerConfig
	var found bool

	if handlerConfig, ok := r.handlers[id]; ok {
		config = handlerConfig
		found = true
	} else {
		// Try prefix matching for dynamic IDs
		for key, handlerConfig := range r.handlers {
			if strings.HasPrefix(id, key) {
				config = handlerConfig
				found = true
				break
			}
		}
	}

	if !found {
		if r.logger != nil {
			r.logger.Warn("No handler found for interaction", slog.String("id", id))
		}
		return
	}

	if r.logger != nil {
		r.logger.Info("✅ Handler found, checking permissions",
			slog.String("id", id),
			slog.Bool("requires_setup", config.RequiresSetup),
			slog.Int("required_permission", int(config.RequiredPermission)))
	}

	// Check permissions before executing handler
	if !r.checkPermissions(ctx, s, i, config) {
		if r.logger != nil {
			r.logger.Warn("❌ Permission check failed", slog.String("id", id))
		}
		return
	}

	if r.logger != nil {
		r.logger.Info("✅ Permission check passed, executing handler", slog.String("id", id))
	}

	// Execute the handler
	config.Handler(ctx, i)
}

func (r *Registry) checkPermissions(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config HandlerConfig) bool {
	if r.logger != nil {
		r.logger.Info("🔍 Starting permission check",
			slog.String("guild_id", i.GuildID),
			slog.Bool("requires_setup", config.RequiresSetup),
			slog.Int("required_permission", int(config.RequiredPermission)))
	}

	// Allow frolf-setup command without permission checking (it has Discord admin perms)
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "frolf-setup" {
		if r.logger != nil {
			r.logger.Info("✅ Allowing frolf-setup command")
		}
		return true
	}

	// If no guild ID, allow (DM interactions)
	if i.GuildID == "" {
		if r.logger != nil {
			r.logger.Info("✅ Allowing DM interaction")
		}
		return true
	}

	// Check if setup is required and guild is configured
	if config.RequiresSetup && r.guildConfigResolver != nil {
		isSetup := r.guildConfigResolver.IsGuildSetupComplete(i.GuildID)
		if r.logger != nil {
			r.logger.Info("🏗️ Setup check",
				slog.String("guild_id", i.GuildID),
				slog.Bool("is_setup_complete", isSetup))
		}
		if !isSetup {
			r.sendErrorResponse(s, i, "❌ This server hasn't been set up yet. An admin must run `/frolf-setup` first.")
			return false
		}
	}

	// If no permission required, allow
	if config.RequiredPermission == NoPermissionRequired {
		return true
	}

	// Get guild config for permission checking
	if r.guildConfigResolver == nil {
		if r.logger != nil {
			r.logger.Error("Cannot check permissions: guild config resolver not set")
		}
		r.sendErrorResponse(s, i, "❌ Permission system not available.")
		return false
	}

	guildConfig, err := r.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to get guild config for permission check", slog.String("guild_id", i.GuildID), slog.Any("error", err))
		}
		r.sendErrorResponse(s, i, "❌ Unable to verify permissions. Please try again.")
		return false
	}

	// Check permissions using inline permission checking to avoid import cycles
	if !r.checkGuildPermission(i.Member, guildConfig, config.RequiredPermission) {
		var roleRequired string
		switch config.RequiredPermission {
		case PlayerRequired:
			roleRequired = "Player"
		case EditorRequired:
			roleRequired = "Editor"
		case AdminRequired:
			roleRequired = "Admin"
		}
		r.sendErrorResponse(s, i, "❌ You need the **"+roleRequired+"** role or higher to use this command.")
		return false
	}

	return true
}

// checkGuildPermission checks if a member has the required permission level
// This is a simplified version to avoid import cycles
func (r *Registry) checkGuildPermission(member *discordgo.Member, guildConfig *storage.GuildConfig, required PermissionLevel) bool {
	if member == nil || guildConfig == nil {
		return false
	}

	// Check for Discord admin permissions first
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	// Get required role ID based on permission level
	var requiredRoleID string
	switch required {
	case PlayerRequired:
		requiredRoleID = guildConfig.RegisteredRoleID
	case EditorRequired:
		requiredRoleID = guildConfig.EditorRoleID
	case AdminRequired:
		requiredRoleID = guildConfig.AdminRoleID
	default:
		return true // No permission required
	}

	if requiredRoleID == "" {
		return false // Role not configured
	}

	// Check if user has the required role or higher
	hasRole := func(roleID string) bool {
		for _, memberRoleID := range member.Roles {
			if memberRoleID == roleID {
				return true
			}
		}
		return false
	}

	// Check for Admin role (highest)
	if guildConfig.AdminRoleID != "" && hasRole(guildConfig.AdminRoleID) {
		return true
	}

	// For Editor permission, also allow Editor role
	if required == EditorRequired && guildConfig.EditorRoleID != "" && hasRole(guildConfig.EditorRoleID) {
		return true
	}

	// For Player permission, allow Player, Editor, or Admin role
	if required == PlayerRequired {
		if (guildConfig.RegisteredRoleID != "" && hasRole(guildConfig.RegisteredRoleID)) ||
			(guildConfig.EditorRoleID != "" && hasRole(guildConfig.EditorRoleID)) ||
			(guildConfig.AdminRoleID != "" && hasRole(guildConfig.AdminRoleID)) {
			return true
		}
	}

	return false
}

func (r *Registry) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	if s == nil || i == nil || i.Interaction == nil {
		if r.logger != nil {
			r.logger.Error("Cannot send error response: session or interaction is nil")
		}
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil && r.logger != nil {
		r.logger.Error("Failed to send error response", slog.Any("error", err))
	}
}
