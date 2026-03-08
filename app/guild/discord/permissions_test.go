package discord

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// helper to build a guild config marked configured
func configuredConfig(player, editor, admin string) *storage.GuildConfig {
	// Provide all required fields so IsConfigured() returns true
	return &storage.GuildConfig{
		GuildID:              "g1",
		SignupChannelID:      "signup-channel",
		EventChannelID:       "event-channel",
		LeaderboardChannelID: "leaderboard-channel",
		RegisteredRoleID:     player,
		EditorRoleID:         editor,
		AdminRoleID:          admin,
	}
}

func TestCheckGuildPermission(t *testing.T) {
	cfg := configuredConfig("player-role", "editor-role", "admin-role")

	tests := []struct {
		name   string
		member *discordgo.Member
		level  Level
		want   bool
	}{
		{"nil config", &discordgo.Member{Roles: []string{"player-role"}}, Player, false},
		{"nil member", nil, Player, false},
		{"player has player", &discordgo.Member{Roles: []string{"player-role"}}, Player, true},
		{"player satisfies player when editor", &discordgo.Member{Roles: []string{"editor-role"}}, Player, true},
		{"player satisfies player when admin", &discordgo.Member{Roles: []string{"admin-role"}}, Player, true},
		{"editor requires editor or admin - editor", &discordgo.Member{Roles: []string{"editor-role"}}, Editor, true},
		{"editor requires editor or admin - admin", &discordgo.Member{Roles: []string{"admin-role"}}, Editor, true},
		{"editor rejected for player only", &discordgo.Member{Roles: []string{"player-role"}}, Editor, false},
		{"admin only admin accepted - admin", &discordgo.Member{Roles: []string{"admin-role"}}, Admin, true},
		{"admin rejected editor", &discordgo.Member{Roles: []string{"editor-role"}}, Admin, false},
		{"admin rejected player", &discordgo.Member{Roles: []string{"player-role"}}, Admin, false},
	}

	for _, tt := range tests {
		// nil config explicit case
		if tt.name == "nil config" {
			if got := CheckGuildPermission(tt.member, nil, tt.level); got != tt.want {
				t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
			}
			continue
		}
		if got := CheckGuildPermission(tt.member, cfg, tt.level); got != tt.want {
			// show roles for clarity
			var roles []string
			if tt.member != nil {
				roles = tt.member.Roles
			}
			t.Errorf("%s: roles=%v level=%v got %v want %v", tt.name, roles, tt.level, got, tt.want)
		}
	}
}

func TestCheckDiscordAdminPermission(t *testing.T) {
	memberAdmin := &discordgo.Member{Permissions: discordgo.PermissionAdministrator}
	memberNonAdmin := &discordgo.Member{Permissions: discordgo.PermissionManageChannels}

	tests := []struct {
		name string
		m    *discordgo.Member
		want bool
	}{
		{"nil member", nil, false},
		{"non admin", memberNonAdmin, false},
		{"admin", memberAdmin, true},
	}

	for _, tt := range tests {
		if got := CheckDiscordAdminPermission(tt.m); got != tt.want {
			permissions := int64(0)
			if tt.m != nil {
				permissions = tt.m.Permissions
			}
			// printing raw perms helps debugging differences
			l := Level(0)
			_ = l // silence unused if future modifications remove Level here
			t.Errorf("%s: got %v want %v perms=%d", tt.name, got, tt.want, permissions)
		}
	}
}

func TestGetUserPermissionLevel(t *testing.T) {
	cfg := configuredConfig("player-role", "editor-role", "admin-role")
	memberPlayer := &discordgo.Member{Roles: []string{"player-role"}}
	memberEditor := &discordgo.Member{Roles: []string{"editor-role"}}
	memberAdmin := &discordgo.Member{Roles: []string{"admin-role"}}
	memberMixed := &discordgo.Member{Roles: []string{"player-role", "editor-role"}}
	memberMixedAdmin := &discordgo.Member{Roles: []string{"player-role", "admin-role"}}
	memberNone := &discordgo.Member{Roles: []string{"other"}}

	// Expected semantics (top-down): Admin > Editor > Player > none
	tests := []struct {
		name   string
		member *discordgo.Member
		want   Level
	}{
		{"admin only", memberAdmin, Admin},
		{"editor only", memberEditor, Editor},
		{"player only", memberPlayer, Player},
		{"mixed player+editor -> editor", memberMixed, Editor},
		{"mixed player+admin -> admin", memberMixedAdmin, Admin},
		{"no relevant roles", memberNone, NoPermission},
		{"nil member", nil, NoPermission},
	}

	for _, tt := range tests {
		got := GetUserPermissionLevel(tt.member, cfg)
		if got != tt.want {
			var roles []string
			if tt.member != nil {
				roles = tt.member.Roles
			}
			// Intentionally failing now if implementation differs.
			// This codifies desired behavior before implementation change (TDD style).
			t.Errorf("%s: roles=%v got %v want %v", tt.name, roles, got, tt.want)
		}
	}
}

func TestPermissionLevelConsistency(t *testing.T) {
	cfg := configuredConfig("player-role", "editor-role", "admin-role")
	members := []*discordgo.Member{
		{Roles: []string{"player-role"}},
		{Roles: []string{"editor-role"}},
		{Roles: []string{"admin-role"}},
		{Roles: []string{"player-role", "editor-role"}},
		{Roles: []string{"player-role", "admin-role"}},
		nil,
	}
	for i, m := range members {
		level := GetUserPermissionLevel(m, cfg)
		if level >= 0 { // only check meaningful levels
			if !CheckGuildPermission(m, cfg, level) {
				var roles []string
				if m != nil {
					roles = m.Roles
				}
				t.Errorf("member %d: level=%v not actually permitted roles=%v", i, level, roles)
			}
		}
	}
}
