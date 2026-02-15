package scorecardupload

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
)

type fakeScorecardUploadManager struct {
	mu sync.Mutex

	buttonCalls int
	modalCalls  int
	fileCalls   int

	lastSession discord.Session
	lastMsg     *discordgo.MessageCreate
}

func (m *fakeScorecardUploadManager) HandleScorecardUploadButton(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	m.mu.Lock()
	m.buttonCalls++
	m.mu.Unlock()
	return ScorecardUploadOperationResult{Success: "ok"}, nil
}

func (m *fakeScorecardUploadManager) HandleScorecardUploadModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	m.mu.Lock()
	m.modalCalls++
	m.mu.Unlock()
	return ScorecardUploadOperationResult{Success: "ok"}, nil
}

func (m *fakeScorecardUploadManager) HandleFileUploadMessage(s discord.Session, msg *discordgo.MessageCreate) {
	m.mu.Lock()
	m.fileCalls++
	m.lastSession = s
	m.lastMsg = msg
	m.mu.Unlock()
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
	registry.SetGuildConfigResolver(&testutils.FakeGuildConfigResolver{
		GetGuildConfigFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
			return &storage.GuildConfig{
				GuildID:              guildID,
				SignupChannelID:      "signup",
				EventChannelID:       "events",
				LeaderboardChannelID: "leaderboard",
				RegisteredRoleID:     "player",
			}, nil
		},
	})
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	messageRegistry := interactions.NewMessageRegistry(logger)
	mgr := &fakeScorecardUploadManager{}

	RegisterHandlers(registry, messageRegistry, mgr)

	// Button press routes via prefix matching.
	buttonInteraction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		ID:      "i1",
		GuildID: "g1",
		Data:    discordgo.MessageComponentInteractionData{CustomID: "round_upload_scorecard|round-123"},
	}}
	buttonInteraction.Member = &discordgo.Member{User: &discordgo.User{ID: "u1"}, Roles: []string{"player"}}
	registry.HandleInteraction(&discordgo.Session{}, buttonInteraction)
	mgr.mu.Lock()
	if mgr.buttonCalls != 1 {
		mgr.mu.Unlock()
		t.Fatalf("expected button handler called once, got %d", mgr.buttonCalls)
	}
	mgr.mu.Unlock()

	// Modal submit routes via prefix matching.
	modalInteraction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		ID:      "i2",
		GuildID: "g1",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "scorecard_upload_modal|round-123"},
	}}
	modalInteraction.Member = &discordgo.Member{User: &discordgo.User{ID: "u1"}, Roles: []string{"player"}}
	registry.HandleInteraction(&discordgo.Session{}, modalInteraction)
	mgr.mu.Lock()
	if mgr.modalCalls != 1 {
		mgr.mu.Unlock()
		t.Fatalf("expected modal handler called once, got %d", mgr.modalCalls)
	}
	mgr.mu.Unlock()

	// MessageCreate handler is wired through MessageRegistry.
	fakeSession := discord.NewFakeSession()
	wrapper := discord.Session(fakeSession)

	adder := &testDiscordgoAdder{}
	messageRegistry.RegisterWithSession(adder, wrapper)
	if adder.handler == nil {
		t.Fatalf("expected MessageCreate handler to be registered")
	}

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "m1",
		ChannelID: "c1",
		Author:    &discordgo.User{ID: "u1"},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.csv"},
		},
	}}
	adder.handler(&discordgo.Session{}, msg)

	deadline := time.Now().Add(1 * time.Second)
	for {
		mgr.mu.Lock()
		fileCalls := mgr.fileCalls
		lastSession := mgr.lastSession
		lastMsg := mgr.lastMsg
		mgr.mu.Unlock()

		if fileCalls == 1 {
			if lastSession != wrapper {
				t.Fatalf("expected wrapper session passed to file handler")
			}
			if lastMsg != msg {
				t.Fatalf("expected message pointer passed through")
			}
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected file handler called once, got %d", fileCalls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRegisterHandlers_ignoresInvalidPayloads(t *testing.T) {
	registry := interactions.NewRegistry()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	messageRegistry := interactions.NewMessageRegistry(logger)
	mgr := &fakeScorecardUploadManager{}

	RegisterHandlers(registry, messageRegistry, mgr)

	// Missing user info should be ignored by registered interaction handlers.
	registry.HandleInteraction(&discordgo.Session{}, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		ID:   "i1",
		Data: discordgo.MessageComponentInteractionData{CustomID: "round_upload_scorecard|round-123"},
	}})
	registry.HandleInteraction(&discordgo.Session{}, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		ID:   "i2",
		Data: discordgo.ModalSubmitInteractionData{CustomID: "scorecard_upload_modal|round-123"},
	}})

	// Invalid MessageCreate payloads should be ignored before entering dispatcher queue.
	fakeSession := discord.NewFakeSession()
	adder := &testDiscordgoAdder{}
	messageRegistry.RegisterWithSession(adder, fakeSession)
	if adder.handler == nil {
		t.Fatalf("expected MessageCreate handler to be registered")
	}
	adder.handler(&discordgo.Session{}, nil)
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{})
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{ID: "m1", ChannelID: "c1"}})

	time.Sleep(50 * time.Millisecond)

	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if mgr.buttonCalls != 0 {
		t.Fatalf("expected button handler to be skipped, got %d calls", mgr.buttonCalls)
	}
	if mgr.modalCalls != 0 {
		t.Fatalf("expected modal handler to be skipped, got %d calls", mgr.modalCalls)
	}
	if mgr.fileCalls != 0 {
		t.Fatalf("expected file handler to be skipped, got %d calls", mgr.fileCalls)
	}
}

func TestRegisterHandlers_skipsNonScorecardMessages(t *testing.T) {
	registry := interactions.NewRegistry()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	messageRegistry := interactions.NewMessageRegistry(logger)
	mgr := &fakeScorecardUploadManager{}

	RegisterHandlers(registry, messageRegistry, mgr)

	fakeSession := discord.NewFakeSession()
	adder := &testDiscordgoAdder{}
	messageRegistry.RegisterWithSession(adder, fakeSession)
	if adder.handler == nil {
		t.Fatalf("expected MessageCreate handler to be registered")
	}

	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:          "m1",
		ChannelID:   "c1",
		Author:      &discordgo.User{ID: "u1"},
		Content:     "hello world",
		Attachments: nil,
	}})
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "m2",
		ChannelID: "c1",
		Author:    &discordgo.User{ID: "u1"},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "photo.png"},
		},
	}})
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "m3",
		ChannelID: "c1",
		Author:    &discordgo.User{ID: "bot", Bot: true},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.csv"},
		},
	}})

	time.Sleep(50 * time.Millisecond)

	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if mgr.fileCalls != 0 {
		t.Fatalf("expected non-scorecard messages to be skipped, got %d calls", mgr.fileCalls)
	}
}
