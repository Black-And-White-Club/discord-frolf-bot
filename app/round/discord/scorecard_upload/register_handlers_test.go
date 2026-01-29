package scorecardupload

import (
	"context"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
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

func (m *fakeScorecardUploadManager) SendUploadError(ctx context.Context, channelID, errorMsg, messageID string) error {
	return nil
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
	registry := interactions.NewRegistry()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	messageRegistry := interactions.NewMessageRegistry(logger)
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
	fakeSession := discord.NewFakeSession()
	wrapper := discord.Session(fakeSession)

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
