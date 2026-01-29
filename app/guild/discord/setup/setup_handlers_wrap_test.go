package setup

import (
	"context"
	"errors"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

// localSetupManager is a minimal stub implementing SetupManager to avoid import cycles in tests.
type localSetupManager struct {
	setupCalled int
	modalCalled int
}

func (l *localSetupManager) HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	l.setupCalled++
	return nil
}

func (l *localSetupManager) SendSetupModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	return nil
}

func (l *localSetupManager) HandleSetupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) error {
	l.modalCalled++
	return nil
}

func TestRegisterHandlers_WiresManager(t *testing.T) {
	reg := interactions.NewRegistry()
	lm := &localSetupManager{}

	RegisterHandlers(reg, lm)

	// 1) Slash command: frolf-setup
	slash := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionApplicationCommand,
		Data: discordgo.ApplicationCommandInteractionData{Name: "frolf-setup"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, slash)

	// 2) Modal submit: guild_setup_modal
	modal := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:   uuid.New().String(),
		Type: discordgo.InteractionModalSubmit,
		Data: discordgo.ModalSubmitInteractionData{CustomID: "guild_setup_modal"},
	}}
	reg.HandleInteraction(&discordgo.Session{}, modal)

	if lm.setupCalled != 1 || lm.modalCalled != 1 {
		t.Fatalf("expected handlers called once each, got setup=%d modal=%d", lm.setupCalled, lm.modalCalled)
	}
}

func TestHandleSetupCommand_Paths(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	// Happy path should send a modal response via SendSetupModal
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		return nil
	}

	sm := &setupManager{
		session:          fakeSession,
		logger:           discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error { return fn(ctx) },
	}

	slash := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String(), Type: discordgo.InteractionApplicationCommand, GuildID: "g1"}}
	if err := sm.HandleSetupCommand(context.Background(), slash); err != nil {
		t.Fatalf("HandleSetupCommand happy path error: %v", err)
	}

	// Nil interaction should error
	if err := sm.HandleSetupCommand(context.Background(), &discordgo.InteractionCreate{}); err == nil {
		t.Fatalf("expected error for nil interaction, got none")
	}
}

func TestNewSetupManager_Constructs(t *testing.T) {
	mgr, err := NewSetupManager(
		nil,                      // session
		nil,                      // publisher
		nil,                      // logger
		nil,                      // helper
		nil,                      // config
		nil,                      // interactionStore
		otel.Tracer("test"),      // tracer (no-op)
		discordmetrics.NewNoop(), // metrics
		nil,                      // guildConfigResolver
	)
	if err != nil {
		t.Fatalf("NewSetupManager returned error: %v", err)
	}
	if mgr == nil {
		t.Fatalf("NewSetupManager returned nil manager")
	}
}

func Test_wrapSetupOperation_SuccessAndError(t *testing.T) {
	tracer := otel.Tracer("test")
	metrics := discordmetrics.NewNoop()

	// Success branch
	if err := wrapSetupOperation(context.Background(), "ok", func(ctx context.Context) error { return nil }, nil, tracer, metrics); err != nil {
		t.Fatalf("wrapSetupOperation success returned error: %v", err)
	}

	// Error branch
	want := errors.New("boom")
	if err := wrapSetupOperation(context.Background(), "err", func(ctx context.Context) error { return want }, nil, tracer, metrics); !errors.Is(err, want) {
		t.Fatalf("wrapSetupOperation error mismatch: got %v want %v", err, want)
	}
}

func TestSendFollowupSuccess_AllFields(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	fakeSession.FollowupMessageCreateFunc = func(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{ID: "ok"}, nil
	}

	sm := &setupManager{session: fakeSession}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String()}}
	res := &SetupResult{
		EventChannelID:       "e",
		LeaderboardChannelID: "l",
		SignupChannelID:      "s",
		UserRoleID:           "ru",
		EditorRoleID:         "re",
		AdminRoleID:          "ra",
	}
	if err := sm.sendFollowupSuccess(i, res); err != nil {
		t.Fatalf("sendFollowupSuccess error: %v", err)
	}
}
