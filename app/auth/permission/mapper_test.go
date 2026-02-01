package permission

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

func TestMapper_MapMemberRole(t *testing.T) {
	mapper := NewMapper()

	tests := []struct {
		name        string
		member      *discordgo.Member
		guildConfig *storage.GuildConfig
		wantRole    PWARole
	}{
		{
			name:        "nil member returns viewer",
			member:      nil,
			guildConfig: &storage.GuildConfig{},
			wantRole:    RoleViewer,
		},
		{
			name: "nil guild config returns viewer",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"some-role"},
			},
			guildConfig: nil,
			wantRole:    RoleViewer,
		},
		{
			name: "member with admin permission gets editor",
			member: &discordgo.Member{
				User:        &discordgo.User{ID: "user-123"},
				Permissions: discordgo.PermissionAdministrator,
				Roles:       []string{},
			},
			guildConfig: &storage.GuildConfig{},
			wantRole:    RoleEditor,
		},
		{
			name: "member with admin role gets editor",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"admin-role-id"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "admin-role-id",
				EditorRoleID:     "editor-role-id",
				RegisteredRoleID: "player-role-id",
			},
			wantRole: RoleEditor,
		},
		{
			name: "member with editor role gets editor",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"editor-role-id"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "admin-role-id",
				EditorRoleID:     "editor-role-id",
				RegisteredRoleID: "player-role-id",
			},
			wantRole: RoleEditor,
		},
		{
			name: "member with registered role gets player",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"player-role-id"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "admin-role-id",
				EditorRoleID:     "editor-role-id",
				RegisteredRoleID: "player-role-id",
			},
			wantRole: RolePlayer,
		},
		{
			name: "member with no matching role gets viewer",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"some-other-role"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "admin-role-id",
				EditorRoleID:     "editor-role-id",
				RegisteredRoleID: "player-role-id",
			},
			wantRole: RoleViewer,
		},
		{
			name: "member with multiple roles gets highest - admin",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"player-role-id", "admin-role-id"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "admin-role-id",
				EditorRoleID:     "editor-role-id",
				RegisteredRoleID: "player-role-id",
			},
			wantRole: RoleEditor, // Admin role maps to editor
		},
		{
			name: "empty role IDs in config returns viewer",
			member: &discordgo.Member{
				User:  &discordgo.User{ID: "user-123"},
				Roles: []string{"some-role"},
			},
			guildConfig: &storage.GuildConfig{
				AdminRoleID:      "",
				EditorRoleID:     "",
				RegisteredRoleID: "",
			},
			wantRole: RoleViewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapper.MapMemberRole(tt.member, tt.guildConfig)
			if got != tt.wantRole {
				t.Errorf("MapMemberRole() = %v, want %v", got, tt.wantRole)
			}
		})
	}
}

func TestPWARole_String(t *testing.T) {
	tests := []struct {
		role PWARole
		want string
	}{
		{RoleViewer, "viewer"},
		{RolePlayer, "player"},
		{RoleEditor, "editor"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("PWARole.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPWARole_IsValid(t *testing.T) {
	tests := []struct {
		role  PWARole
		valid bool
	}{
		{RoleViewer, true},
		{RolePlayer, true},
		{RoleEditor, true},
		{PWARole("admin"), false},
		{PWARole(""), false},
		{PWARole("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.valid {
				t.Errorf("PWARole.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}
