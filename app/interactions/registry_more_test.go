package interactions

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

func Test_checkGuildPermission_nilInputs_ReturnsFalse(t *testing.T) {
	r := NewRegistry()
	if r.checkGuildPermission(nil, &storage.GuildConfig{}, PlayerRequired) {
		t.Fatalf("expected false for nil member")
	}
	if r.checkGuildPermission(&discordgo.Member{}, nil, PlayerRequired) {
		t.Fatalf("expected false for nil guild config")
	}
}

func Test_checkGuildPermission_roleNotConfigured_ReturnsFalse(t *testing.T) {
	r := NewRegistry()
	cfg := &storage.GuildConfig{
		RegisteredRoleID: "", // not configured
		EditorRoleID:     "",
		AdminRoleID:      "",
	}
	m := &discordgo.Member{Roles: []string{"whatever"}}
	if r.checkGuildPermission(m, cfg, PlayerRequired) {
		t.Fatalf("expected false when required role id is not configured")
	}
	if r.checkGuildPermission(m, cfg, EditorRequired) {
		t.Fatalf("expected false when required role id is not configured")
	}
	if r.checkGuildPermission(m, cfg, AdminRequired) {
		t.Fatalf("expected false when required role id is not configured")
	}
}
