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
	// Initialize context. In a production environment, you might extract
	// trace headers from the interaction if provided by a proxy.
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

	var config HandlerConfig
	var found bool

	if handlerConfig, ok := r.handlers[id]; ok {
		config = handlerConfig
		found = true
	} else {
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

	// Check permissions and server setup status
	if !r.checkPermissions(ctx, s, i, config) {
		return
	}

	// Execute the handler with context
	config.Handler(ctx, i)
}

func (r *Registry) checkPermissions(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config HandlerConfig) bool {
	// 1. bypass: Allow frolf-setup command (uses Discord built-in Admin perms usually)
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "frolf-setup" {
		return true
	}

	// 2. bypass: DM interactions (no guild context)
	if i.GuildID == "" {
		return true
	}

	// 3. Setup Check: If command requires setup, verify via resolver
	if config.RequiresSetup && r.guildConfigResolver != nil {
		isSetup := r.guildConfigResolver.IsGuildSetupComplete(i.GuildID)
		if !isSetup {
			r.sendErrorResponse(s, i, "❌ This server hasn't been set up yet. An admin must run `/frolf-setup` first.")
			return false
		}
	}

	// 4. Permission Check: If level is NoPermissionRequired, we are done
	if config.RequiredPermission == NoPermissionRequired {
		return true
	}

	if r.guildConfigResolver == nil {
		r.sendErrorResponse(s, i, "❌ Permission system not available.")
		return false
	}

	// Fetch guild config using the context-aware method
	guildConfig, err := r.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to get guild config for permission check",
				slog.String("guild_id", i.GuildID),
				slog.Any("error", err))
		}

		// Handle specific loading state for better UX
		if guildconfig.IsConfigLoading(err) {
			r.sendErrorResponse(s, i, "⏳ Server configuration is still loading from the database. Please try again in a few seconds.")
			return false
		}

		r.sendErrorResponse(s, i, "❌ Unable to verify your permissions at this time.")
		return false
	}

	// 5. Evaluate Roles
	if !r.checkGuildPermission(i.Member, guildConfig, config.RequiredPermission) {
		roleName := "Player"
		switch config.RequiredPermission {
		case EditorRequired:
			roleName = "Editor"
		case AdminRequired:
			roleName = "Admin"
		}
		r.sendErrorResponse(s, i, "❌ You need the **"+roleName+"** role or higher to use this.")
		return false
	}

	return true
}

func (r *Registry) checkGuildPermission(member *discordgo.Member, guildConfig *storage.GuildConfig, required PermissionLevel) bool {
	if member == nil || guildConfig == nil {
		return false
	}

	// Discord Administrators bypass all bot-level role checks
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	hasRole := func(roleID string) bool {
		if roleID == "" {
			return false
		}
		for _, mRoleID := range member.Roles {
			if mRoleID == roleID {
				return true
			}
		}
		return false
	}

	// Hierarchy check:
	// Admin role satisfies all levels
	if hasRole(guildConfig.AdminRoleID) {
		return true
	}

	// Editor role satisfies Editor and Player levels
	if required == EditorRequired || required == PlayerRequired {
		if hasRole(guildConfig.EditorRoleID) {
			return true
		}
	}

	// Registered role satisfies Player level
	if required == PlayerRequired {
		if hasRole(guildConfig.RegisteredRoleID) {
			return true
		}
	}

	return false
}

func (r *Registry) sendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	if s == nil || i == nil || i.Interaction == nil {
		return
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
