package permission

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

type PWARole string

const (
	RoleViewer PWARole = "viewer"
	RolePlayer PWARole = "player"
	RoleEditor PWARole = "editor"
)

// Mapper maps Discord member roles to PWA permission levels
type Mapper interface {
	// MapMemberRole returns the highest PWA role for a Discord member
	MapMemberRole(member *discordgo.Member, guildConfig *storage.GuildConfig) PWARole
}

type mapper struct{}

// NewMapper creates a new permission mapper
func NewMapper() Mapper {
	return &mapper{}
}

// MapMemberRole returns the highest PWA role for a Discord member
func (m *mapper) MapMemberRole(member *discordgo.Member, guildConfig *storage.GuildConfig) PWARole {
	if member == nil || guildConfig == nil {
		return RoleViewer
	}

	// Discord Administrators get editor access
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return RoleEditor
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

	// Check roles in descending privilege order
	if hasRole(guildConfig.AdminRoleID) {
		return RoleEditor
	}

	if hasRole(guildConfig.EditorRoleID) {
		return RoleEditor
	}

	if hasRole(guildConfig.RegisteredRoleID) {
		return RolePlayer
	}

	return RoleViewer
}

// String returns the string representation of PWARole
func (r PWARole) String() string {
	return string(r)
}

// IsValid checks if the PWARole is valid
func (r PWARole) IsValid() bool {
	switch r {
	case RoleViewer, RolePlayer, RoleEditor:
		return true
	default:
		return false
	}
}
