package interactions

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

func TestPrefixMatching_HandlerInvoked(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandler("my-action", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true })

	// Modal submit with customID that has a prefix match
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "g1",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "my-action:123"},
	}}

	// nil session is fine as long as permissions pass and no error response is sent
	r.HandleInteraction(nil, i)

	if !called {
		t.Fatalf("expected handler to be called for prefix match")
	}
}

func TestPrefixMatching_LongestPrefixWins(t *testing.T) {
	r := NewRegistry()
	called := ""
	r.RegisterHandler("action|", func(ctx context.Context, i *discordgo.InteractionCreate) { called = "short" })
	r.RegisterHandler("action|special|", func(ctx context.Context, i *discordgo.InteractionCreate) { called = "long" })

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "g1",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "action|special|123"},
	}}

	r.HandleInteraction(nil, i)
	if called != "long" {
		t.Fatalf("expected longest prefix handler, got %q", called)
	}
}

func TestDMInteraction_NotAllowlisted_Blocked(t *testing.T) {
	r := NewRegistry()
	called := false
	// Requires setup+admin and should not run in a non-allowlisted DM interaction.
	r.RegisterHandlerWithPermissions("cmd", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true }, AdminRequired, true)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "", // DM
		Data:    discordgo.ApplicationCommandInteractionData{Name: "cmd"},
	}}

	r.HandleInteraction(nil, i)
	if called {
		t.Fatalf("expected handler to be blocked for non-allowlisted DM interaction")
	}
}

func TestDMInteraction_Allowlisted_BypassesGuildChecks(t *testing.T) {
	r := NewRegistry()
	called := false
	r.RegisterHandlerWithPermissions("signup_modal", func(ctx context.Context, i *discordgo.InteractionCreate) { called = true }, AdminRequired, true)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "signup_modal|guild_id=g1"},
	}}

	r.HandleInteraction(nil, i)
	if !called {
		t.Fatalf("expected allowlisted DM interaction to execute")
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
	delay         time.Duration
	getCalls      int
}

func (s *stubResolver) GetGuildConfigWithContext(ctx context.Context, _ string) (*storage.GuildConfig, error) {
	s.getCalls++
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.cfg, nil
}
func (s *stubResolver) RequestGuildConfigAsync(_ context.Context, _ string) {}
func (s *stubResolver) IsGuildSetupComplete(_ string) bool                  { return s.setupComplete }
func (s *stubResolver) HandleGuildConfigReceived(_ context.Context, _ string, _ *storage.GuildConfig) {
}
func (s *stubResolver) ClearInflightRequest(_ context.Context, _ string)        {}
func (s *stubResolver) HandleBackendError(_ context.Context, _ string, _ error) {}

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
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "g1",
		Data:    discordgo.MessageComponentInteractionData{CustomID: "comp-42"},
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

func TestRequiresSetupAndPermission_UsesSingleGuildConfigFetch(t *testing.T) {
	r := NewRegistry()
	executed := false
	r.RegisterHandlerWithPermissions("needs-setup-and-player", func(ctx context.Context, i *discordgo.InteractionCreate) {
		executed = true
	}, PlayerRequired, true)

	resolver := &stubResolver{
		cfg: &storage.GuildConfig{
			GuildID:              "g1",
			SignupChannelID:      "signup",
			EventChannelID:       "event",
			LeaderboardChannelID: "leaderboard",
			RegisteredRoleID:     "player",
		},
	}
	r.SetGuildConfigResolver(resolver)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Member:  &discordgo.Member{Roles: []string{"player"}},
		Data:    discordgo.ApplicationCommandInteractionData{Name: "needs-setup-and-player"},
	}}

	r.HandleInteraction(nil, i)

	if !executed {
		t.Fatalf("expected handler to execute")
	}
	if resolver.getCalls != 1 {
		t.Fatalf("expected exactly one guild config fetch, got %d", resolver.getCalls)
	}
}

func TestPermissionLookupTimeout_BlocksHandler(t *testing.T) {
	r := NewRegistry()
	executed := false
	r.RegisterHandlerWithPermissions("slow-config", func(ctx context.Context, i *discordgo.InteractionCreate) {
		executed = true
	}, PlayerRequired, false)

	resolver := &stubResolver{
		delay: 2 * time.Second,
		cfg: &storage.GuildConfig{
			GuildID:          "g1",
			RegisteredRoleID: "player",
		},
	}
	r.SetGuildConfigResolver(resolver)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Member:  &discordgo.Member{Roles: []string{"player"}},
		Data:    discordgo.ApplicationCommandInteractionData{Name: "slow-config"},
	}}

	r.HandleInteraction(nil, i)

	if executed {
		t.Fatalf("handler should not execute when guild config lookup times out")
	}
	if resolver.getCalls != 1 {
		t.Fatalf("expected one lookup attempt, got %d", resolver.getCalls)
	}
}

func TestRegisterMutatingHandler_BlockingPolicies(t *testing.T) {
	tests := []struct {
		name     string
		resolver *stubResolver
		member   *discordgo.Member
		wantRun  bool
	}{
		{
			name: "setup incomplete is blocked",
			resolver: &stubResolver{
				cfg: &storage.GuildConfig{GuildID: "g1"},
			},
			member:  &discordgo.Member{Roles: []string{"player"}},
			wantRun: false,
		},
		{
			name: "missing required role is blocked",
			resolver: &stubResolver{
				cfg: &storage.GuildConfig{
					GuildID:              "g1",
					SignupChannelID:      "signup",
					EventChannelID:       "event",
					LeaderboardChannelID: "leaderboard",
					RegisteredRoleID:     "player",
				},
			},
			member:  &discordgo.Member{Roles: []string{"viewer"}},
			wantRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			r.SetGuildConfigResolver(tt.resolver)

			executed := false
			r.RegisterMutatingHandler("mutate", func(ctx context.Context, i *discordgo.InteractionCreate) {
				executed = true
			}, MutatingHandlerPolicy{RequiredPermission: PlayerRequired, RequiresSetup: true})

			i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type:    discordgo.InteractionApplicationCommand,
				GuildID: "g1",
				Member:  tt.member,
				Data:    discordgo.ApplicationCommandInteractionData{Name: "mutate"},
			}}

			r.HandleInteraction(nil, i)

			if executed != tt.wantRun {
				t.Fatalf("executed=%v want=%v", executed, tt.wantRun)
			}
		})
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

func TestHandleInteraction_NilPayload_NoPanic(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())

	r.HandleInteraction(nil, nil)
	r.HandleInteraction(nil, &discordgo.InteractionCreate{})
}

func TestHandleInteraction_HandlerPanic_Recovered(t *testing.T) {
	r := NewRegistry()
	r.SetLogger(slog.Default())
	called := false
	r.RegisterHandler("panic-cmd", func(ctx context.Context, i *discordgo.InteractionCreate) {
		called = true
		panic("boom")
	})

	r.HandleInteraction(nil, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "g1",
		Data:    discordgo.ApplicationCommandInteractionData{Name: "panic-cmd"},
	}})

	if !called {
		t.Fatalf("expected handler to run before panic")
	}
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

	// Payload admin bit alone should not bypass role checks.
	if r.checkGuildPermission(&discordgo.Member{Permissions: discordgo.PermissionAdministrator}, cfg, AdminRequired) {
		t.Fatalf("admin permission bit alone should not satisfy admin required")
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
