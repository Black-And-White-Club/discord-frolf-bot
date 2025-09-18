package discord

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// Level represents permission levels for guild commands
type Level int

const (
	Player Level = iota
	Editor
	Admin
	// NoPermission represents absence of any mapped role
	NoPermission Level = -1
)

// CheckGuildPermission checks if a member has the required permission level
// based on guild-configured roles
func CheckGuildPermission(member *discordgo.Member, guildConfig *storage.GuildConfig, required Level) bool {
	if guildConfig == nil || !guildConfig.IsConfigured() {
		return false
	}

	if member == nil {
		return false
	}

	// Check user's roles against guild config
	for _, roleID := range member.Roles {
		switch required {
		case Player:
			if roleID == guildConfig.RegisteredRoleID ||
				roleID == guildConfig.EditorRoleID ||
				roleID == guildConfig.AdminRoleID {
				return true
			}
		case Editor:
			if roleID == guildConfig.EditorRoleID ||
				roleID == guildConfig.AdminRoleID {
				return true
			}
		case Admin:
			if roleID == guildConfig.AdminRoleID {
				return true
			}
		}
	}

	return false
}

// CheckDiscordAdminPermission checks for Discord's built-in admin permissions
// Used for commands like /frolf-setup that need to work before guild roles exist
func CheckDiscordAdminPermission(member *discordgo.Member) bool {
	if member == nil {
		return false
	}
	return (member.Permissions & discordgo.PermissionAdministrator) != 0
}

// GetUserPermissionLevel returns the highest permission level a user has
func GetUserPermissionLevel(member *discordgo.Member, guildConfig *storage.GuildConfig) Level {
	// Evaluate from highest to lowest privilege.
	if CheckGuildPermission(member, guildConfig, Admin) {
		return Admin
	}
	if CheckGuildPermission(member, guildConfig, Editor) {
		return Editor
	}
	if CheckGuildPermission(member, guildConfig, Player) {
		return Player
	}
	return NoPermission
}

// PermissionLevelString returns a string representation of the permission level
func (l Level) String() string {
	switch l {
	case Player:
		return "Player"
	case Editor:
		return "Editor"
	case Admin:
		return "Admin"
	default:
		return "Unknown"
	}
}
