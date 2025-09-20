package scoreround

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/bwmarrin/discordgo"
)

// canOverrideFinalized encapsulates logic determining if a member can override finalized scores.
func canOverrideFinalized(member *discordgo.Member, cfg *config.Config) bool {
	if member == nil || member.User == nil {
		return false
	}
	// 1. Check configured admin role ID
	if cfg != nil && cfg.Discord.AdminRoleID != "" {
		for _, r := range member.Roles {
			if r == cfg.Discord.AdminRoleID {
				return true
			}
		}
	}
	// 2. Check permission bits (Administrator or Manage Server)
	if member.Permissions != 0 {
		perms := member.Permissions
		if (perms&discordgo.PermissionAdministrator) != 0 || (perms&discordgo.PermissionManageServer) != 0 {
			return true
		}
	}
	return false
}
