// interactions/registry.go
package interactions

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sort"
	"strings"
	"time"

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

const interactionGuildConfigTimeout = 800 * time.Millisecond

type HandlerConfig struct {
	Handler            func(ctx context.Context, i *discordgo.InteractionCreate)
	RequiredPermission PermissionLevel
	RequiresSetup      bool // Whether the guild must be configured to use this command
	IsMutating         bool
}

// MutatingHandlerPolicy defines setup and role requirements for state-changing interactions.
type MutatingHandlerPolicy struct {
	RequiredPermission PermissionLevel
	RequiresSetup      bool
}

type Registry struct {
	handlers            map[string]HandlerConfig
	handlerPrefixes     []string
	dmSafePrefixes      []string
	guildConfigResolver guildconfig.GuildConfigResolver
	logger              *slog.Logger
}

func NewRegistry() *Registry {
	r := &Registry{
		handlers:        make(map[string]HandlerConfig),
		handlerPrefixes: make([]string, 0),
		dmSafePrefixes:  make([]string, 0),
	}

	// Explicitly allow known DM-safe interaction IDs/prefixes.
	r.addDMSafePrefix("signup_button|")
	r.addDMSafePrefix("signup_modal")
	r.addDMSafePrefix("set-udisc-name")

	return r
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
	r.registerHandlerConfig(id, HandlerConfig{
		Handler:            handler,
		RequiredPermission: NoPermissionRequired,
		RequiresSetup:      false,
		IsMutating:         false,
	})
}

// RegisterHandlerWithPermissions registers a handler with permission requirements
func (r *Registry) RegisterHandlerWithPermissions(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate), permission PermissionLevel, requiresSetup bool) {
	r.registerHandlerConfig(id, HandlerConfig{
		Handler:            handler,
		RequiredPermission: permission,
		RequiresSetup:      requiresSetup,
		IsMutating:         false,
	})
}

// RegisterMutatingHandler registers a mutating handler with explicit setup/permission policy.
func (r *Registry) RegisterMutatingHandler(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate), policy MutatingHandlerPolicy) {
	r.registerHandlerConfig(id, HandlerConfig{
		Handler:            handler,
		RequiredPermission: policy.RequiredPermission,
		RequiresSetup:      policy.RequiresSetup,
		IsMutating:         true,
	})
}

func (r *Registry) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i == nil || i.Interaction == nil {
		if r.logger != nil {
			r.logger.Warn("Ignoring interaction with nil payload")
		}
		return
	}

	defer r.recoverFromPanic("interaction_dispatch", "", i)

	// Initialize context. In a production environment, you might extract
	// trace headers from the interaction if provided by a proxy.
	ctx := context.Background()
	var id string

	if r.logger != nil {
		r.logger.Info("interaction received",
			slog.String("type", i.Type.String()),
			slog.String("guild_id", i.GuildID),
			slog.String("user_id", interactionUserID(i)),
		)
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		id = i.ApplicationCommandData().Name
		if r.logger != nil {
			r.logger.Info("application command", slog.String("command", id))
		}
	case discordgo.InteractionMessageComponent:
		id = i.MessageComponentData().CustomID
		if r.logger != nil {
			r.logger.Info("message component", slog.String("custom_id", id))
		}
	case discordgo.InteractionModalSubmit:
		modalData := i.ModalSubmitData()
		if modalData.CustomID == "" {
			if r.logger != nil {
				r.logger.Warn("ignoring modal submission with empty custom id")
			}
			return
		}
		id = modalData.CustomID
		if r.logger != nil {
			r.logger.Info("modal submit", slog.String("custom_id", id))
		}
	default:
		if r.logger != nil {
			r.logger.Warn("ignoring unsupported interaction type",
				slog.String("type", i.Type.String()))
		}
		return
	}

	if id == "" {
		if r.logger != nil {
			r.logger.Warn("ignoring interaction with empty id", slog.String("type", i.Type.String()))
		}
		return
	}

	var config HandlerConfig
	var found bool

	if handlerConfig, ok := r.handlers[id]; ok {
		config = handlerConfig
		found = true
	} else {
		for _, key := range r.handlerPrefixes {
			if strings.HasPrefix(id, key) {
				config = r.handlers[key]
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
	if !r.checkPermissions(ctx, s, i, id, config) {
		return
	}

	// Execute the handler with context
	config.Handler(ctx, i)
}

func (r *Registry) checkPermissions(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, interactionID string, config HandlerConfig) bool {
	// 1. bypass: Allow frolf-setup command (uses Discord built-in Admin perms usually)
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "frolf-setup" {
		return true
	}

	// 2. Explicit DM allowlist (no guild context)
	if i.GuildID == "" {
		if r.isDMSafeInteraction(interactionID) {
			return true
		}
		if r.logger != nil {
			r.logger.Warn("blocking non-allowlisted DM interaction", slog.String("id", interactionID))
		}
		r.sendErrorResponse(s, i, "❌ This interaction is only available inside a server.")
		return false
	}

	needsGuildConfig := config.RequiresSetup || config.RequiredPermission != NoPermissionRequired
	if !needsGuildConfig {
		return true
	}

	if r.guildConfigResolver == nil {
		r.sendErrorResponse(s, i, "❌ Permission system not available.")
		return false
	}

	lookupCtx, cancel := context.WithTimeout(ctx, interactionGuildConfigTimeout)
	defer cancel()

	guildConfig, err := r.guildConfigResolver.GetGuildConfigWithContext(lookupCtx, i.GuildID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to get guild config for permission check",
				slog.String("guild_id", i.GuildID),
				slog.Any("error", err))
		}

		// Handle loading and timeout states with a quick retry hint.
		if guildconfig.IsConfigLoading(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			r.sendErrorResponse(s, i, "⏳ Server configuration is still loading. Please try again in a few seconds.")
			return false
		}

		r.sendErrorResponse(s, i, "❌ Unable to verify your permissions at this time.")
		return false
	}

	if guildConfig == nil {
		r.sendErrorResponse(s, i, "⏳ Server configuration is still loading. Please try again in a few seconds.")
		return false
	}

	if config.RequiresSetup && !guildConfig.IsConfigured() {
		r.sendErrorResponse(s, i, "❌ This server hasn't been set up yet. An admin must run `/frolf-setup` first.")
		return false
	}

	// No role check required once setup passes.
	if config.RequiredPermission == NoPermissionRequired {
		return true
	}

	// Evaluate role requirements.
	member := r.resolveInteractionMember(s, i)
	if !r.checkGuildPermission(member, guildConfig, config.RequiredPermission) {
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
	if required == NoPermissionRequired {
		return true
	}

	if member == nil || guildConfig == nil {
		return false
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

func (r *Registry) resolveInteractionMember(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.Member {
	if i == nil {
		return nil
	}

	member := i.Member
	if i.GuildID == "" {
		return member
	}

	if member != nil && len(member.Roles) > 0 {
		return member
	}

	if s == nil {
		return member
	}

	userID := interactionUserID(i)
	if userID == "" {
		return member
	}

	resolved, err := s.GuildMember(i.GuildID, userID)
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("failed to resolve guild member for permission check",
				slog.String("guild_id", i.GuildID),
				slog.String("user_id", userID),
				slog.Any("error", err),
			)
		}
		return member
	}

	if member == nil {
		return resolved
	}

	if member.User == nil {
		member.User = resolved.User
	}
	if len(member.Roles) == 0 {
		member.Roles = resolved.Roles
	}

	return member
}

func (r *Registry) registerHandlerConfig(id string, config HandlerConfig) {
	if id == "" {
		return
	}

	config.Handler = r.wrapHandler(id, config.Handler)

	if _, exists := r.handlers[id]; !exists {
		r.handlerPrefixes = append(r.handlerPrefixes, id)
		sortBySpecificity(r.handlerPrefixes)
	}

	r.handlers[id] = config
}

func (r *Registry) wrapHandler(id string, handler func(ctx context.Context, i *discordgo.InteractionCreate)) func(ctx context.Context, i *discordgo.InteractionCreate) {
	if handler == nil {
		return func(context.Context, *discordgo.InteractionCreate) {}
	}

	return func(ctx context.Context, i *discordgo.InteractionCreate) {
		defer r.recoverFromPanic("interaction_handler", id, i)
		handler(ctx, i)
	}
}

func (r *Registry) recoverFromPanic(scope, handlerID string, i *discordgo.InteractionCreate) {
	if recovered := recover(); recovered != nil && r.logger != nil {
		r.logger.Error("panic recovered",
			slog.String("scope", scope),
			slog.String("handler_id", handlerID),
			slog.String("guild_id", func() string {
				if i == nil {
					return ""
				}
				return i.GuildID
			}()),
			slog.String("user_id", interactionUserID(i)),
			slog.Any("panic", recovered),
			slog.String("stack_trace", string(debug.Stack())),
		)
	}
}

func (r *Registry) addDMSafePrefix(prefix string) {
	if prefix == "" {
		return
	}

	for _, existing := range r.dmSafePrefixes {
		if existing == prefix {
			return
		}
	}

	r.dmSafePrefixes = append(r.dmSafePrefixes, prefix)
	sortBySpecificity(r.dmSafePrefixes)
}

func (r *Registry) isDMSafeInteraction(interactionID string) bool {
	for _, prefix := range r.dmSafePrefixes {
		if strings.HasPrefix(interactionID, prefix) {
			return true
		}
	}
	return false
}

func sortBySpecificity(values []string) {
	sort.SliceStable(values, func(i, j int) bool {
		if len(values[i]) == len(values[j]) {
			return values[i] < values[j]
		}
		return len(values[i]) > len(values[j])
	})
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
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
