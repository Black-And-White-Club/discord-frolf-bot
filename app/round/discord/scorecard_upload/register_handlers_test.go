package scorecardupload

import (
	"context"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

type fakeScorecardUploadManager struct {
	buttonCalls int
	modalCalls  int
	fileCalls   int

	lastSession discord.Session
	lastMsg     *discordgo.MessageCreate
}

func (m *fakeScorecardUploadManager) HandleScorecardUploadButton(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	m.buttonCalls++
	return ScorecardUploadOperationResult{Success: "ok"}, nil
}

func (m *fakeScorecardUploadManager) HandleScorecardUploadModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	m.modalCalls++
	return ScorecardUploadOperationResult{Success: "ok"}, nil
}

func (m *fakeScorecardUploadManager) HandleFileUploadMessage(s discord.Session, msg *discordgo.MessageCreate) {
	m.fileCalls++
	m.lastSession = s
	m.lastMsg = msg
}

type testDiscordgoAdder struct {
	handler func(s *discordgo.Session, e *discordgo.MessageCreate)
}

func (a *testDiscordgoAdder) AddHandler(handler interface{}) func() {
	fn, ok := handler.(func(s *discordgo.Session, e *discordgo.MessageCreate))
	if !ok {
		panic("unexpected handler type")
	}
	a.handler = fn
	return func() {}
}

func TestRegisterHandlers_wiresButtonModalAndMessageHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	registry := interactions.NewRegistry()
	messageRegistry := interactions.NewMessageRegistry()
	mgr := &fakeScorecardUploadManager{}

	RegisterHandlers(registry, messageRegistry, mgr)

	// Button press routes via prefix matching.
	buttonInteraction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		ID:   "i1",
		Data: discordgo.MessageComponentInteractionData{CustomID: "round_upload_scorecard|round-123"},
	}}
	buttonInteraction.Member = &discordgo.Member{User: &discordgo.User{ID: "u1"}}
	registry.HandleInteraction(&discordgo.Session{}, buttonInteraction)
	if mgr.buttonCalls != 1 {
		t.Fatalf("expected button handler called once, got %d", mgr.buttonCalls)
	}

	// Modal submit routes via prefix matching.
	modalInteraction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		ID:   "i2",
		Data: discordgo.ModalSubmitInteractionData{CustomID: "scorecard_upload_modal|round-123"},
	}}
	modalInteraction.Member = &discordgo.Member{User: &discordgo.User{ID: "u1"}}
	registry.HandleInteraction(&discordgo.Session{}, modalInteraction)
	if mgr.modalCalls != 1 {
		t.Fatalf("expected modal handler called once, got %d", mgr.modalCalls)
	}

	// MessageCreate handler is wired through MessageRegistry.
	wrapperSession := discordmocks.NewMockSession(ctrl)
	wrapper := discord.Session(wrapperSession)

	adder := &testDiscordgoAdder{}
	messageRegistry.RegisterWithSession(adder, wrapper)
	if adder.handler == nil {
		t.Fatalf("expected MessageCreate handler to be registered")
	}

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{ID: "m1"}}
	adder.handler(&discordgo.Session{}, msg)
	if mgr.fileCalls != 1 {
		t.Fatalf("expected file handler called once, got %d", mgr.fileCalls)
	}
	if mgr.lastSession != wrapper {
		t.Fatalf("expected wrapper session passed to file handler")
	}
	if mgr.lastMsg != msg {
		t.Fatalf("expected message pointer passed through")
	}
}
