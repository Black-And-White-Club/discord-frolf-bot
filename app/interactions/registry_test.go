package interactions

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

func TestPrefixMatching_HandlerInvoked(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandler("my-action", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true })

	// Modal submit with customID that has a prefix match
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		Data: discordgo.ModalSubmitInteractionData{CustomID: "my-action:123"},
	}}

	// nil session is fine as long as permissions pass and no error response is sent
	r.HandleInteraction(nil, i)

	if !called {
		t.Fatalf("expected handler to be called for prefix match")
	}
}

func TestDMBypass_AllowsRegardlessOfPermissions(t *testing.T) {
	r := NewRegistry()
	called := false
	// Require admin, but DM should bypass
	r.RegisterHandlerWithPermissions("cmd", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true }, AdminRequired, true)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "", // DM
		Data:    discordgo.ApplicationCommandInteractionData{Name: "cmd"},
	}}

	r.HandleInteraction(nil, i)
	if !called {
		t.Fatalf("expected handler to be called for DM interaction")
	}
}

func TestFrolfSetupBypass_AllowsWithoutPermissionChecks(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandlerWithPermissions("frolf-setup", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true }, AdminRequired, true)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "frolf-setup"},
	}}

	r.HandleInteraction(nil, i)
	if !called {
		t.Fatalf("expected handler to be called for frolf-setup bypass")
	}
}

func TestNoPermissionRequired_Allows(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandler("ping", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true })

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "ping"},
	}}

	r.HandleInteraction(nil, i)
	if !called {
		t.Fatalf("expected handler to be called for NoPermissionRequired")
	}
}

// stubResolver allows controlling setup and errors
type stubResolver struct {
	setupComplete bool
	cfg           *storage.GuildConfig
	err           error
}

func (s *stubResolver) GetGuildConfigWithContext(_ context.Context, _ string) (*storage.GuildConfig, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.cfg, nil
}
func (s *stubResolver) IsGuildSetupComplete(_ string) bool { return s.setupComplete }
func (s *stubResolver) HandleGuildConfigReceived(_ context.Context, _ string, _ *storage.GuildConfig) {
}
func (s *stubResolver) ClearInflightRequest(_ context.Context, _ string) {}

func TestRequiresSetup_NotConfigured_SendsError_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	r.RegisterHandlerWithPermissions("needs-setup", func(ctx context.Context, i *discordgo.InteractionCreate) {}, NoPermissionRequired, true)
	r.SetGuildConfigResolver(&stubResolver{setupComplete: false})

	// No session to ensure nil-safe error path
	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "needs-setup"},
	}})
}

func TestResolverError_SendsError_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	r.RegisterHandlerWithPermissions("needs-perm", func(ctx context.Context, i *discordgo.InteractionCreate) {}, PlayerRequired, false)
	r.SetGuildConfigResolver(&stubResolver{setupComplete: true, err: errors.New("boom")})

	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "needs-perm"},
	}})
}

func TestNoHandlerFound_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	// No handlers registered
	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "unknown"},
	}})
}

func TestMessageComponent_PrefixMatching(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandler("comp-", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true })

	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "comp-42"},
	}})

	if !called {
		t.Fatalf("expected prefix handler to be called for message component")
	}
}

func TestModalSubmit_EmptyCustomID_Ignored(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		Data: discordgo.ModalSubmitInteractionData{CustomID: ""},
	}})
}

func TestPermissionDenied_EditorRequired_PlayerOnly_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())

	// Register a handler requiring Editor, ensure it's not executed when member has only player role
	executed := false
	r.RegisterHandlerWithPermissions("needs-editor", func(ctx context.Context, i *discordgo.InteractionCreate) { executed = true }, EditorRequired, false)

	cfg := &storage.GuildConfig{RegisteredRoleID: "player", EditorRoleID: "editor", AdminRoleID: "admin"}
	r.SetGuildConfigResolver(&stubResolver{setupComplete: true, cfg: cfg})

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Member:  &discordgo.Member{Roles: []string{"player"}},
		Data:    discordgo.ApplicationCommandInteractionData{Name: "needs-editor"},
	}}

	// Use nil session; sendErrorResponse is nil-guarded
	r.HandleInteraction(nil, i)
	if executed {
		t.Fatalf("handler should not execute when permission denied")
	}
}

func TestMessageComponent_NoHandlerFound_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	// No handlers registered with matching prefix or exact id
	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{CustomID: "unknown-component"},
	}})
}

func TestCheckGuildPermission_Variants(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())

	cfg := &storage.GuildConfig{
		RegisteredRoleID: "player",
		EditorRoleID:     "editor",
		AdminRoleID:      "admin",
	}

	member := func(roles ...string) *discordgo.Member { return &discordgo.Member{Roles: roles} }

	// Discord admin bit passes immediately
	if !r.checkGuildPermission(&discordgo.Member{Permissions: discordgo.PermissionAdministrator}, cfg, AdminRequired) {
		t.Fatalf("admin bit should pass")
	}

	// AdminRequired: must have admin role
	if r.checkGuildPermission(member("editor"), cfg, AdminRequired) {
		t.Fatalf("editor should not satisfy admin required")
	}
	if !r.checkGuildPermission(member("admin"), cfg, AdminRequired) {
		t.Fatalf("admin role should satisfy admin required")
	}

	// EditorRequired: editor or admin
	if !r.checkGuildPermission(member("editor"), cfg, EditorRequired) || !r.checkGuildPermission(member("admin"), cfg, EditorRequired) {
		t.Fatalf("editor/admin roles should satisfy editor required")
	}
	if r.checkGuildPermission(member("player"), cfg, EditorRequired) {
		t.Fatalf("player should not satisfy editor required")
	}

	// PlayerRequired: player/editor/admin
	if !r.checkGuildPermission(member("player"), cfg, PlayerRequired) || !r.checkGuildPermission(member("editor"), cfg, PlayerRequired) || !r.checkGuildPermission(member("admin"), cfg, PlayerRequired) {
		t.Fatalf("player/editor/admin should satisfy player required")
	}
}

// Note: error-response branches require a live *discordgo.Session;
// we intentionally avoid asserting those here to keep tests unit-level
// and focused on our handler selection and permission logic.
