package interactions

import (
	"context"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

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

func TestMessageRegistry_RegisterWithSession_fansOutToHandlers(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	reg := NewMessageRegistry(slog.Default())

	var called []string
	reg.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		if s != fakeSession {
			t.Fatalf("expected wrapper session to be passed through")
		}
		if m == nil || m.Message == nil || m.Message.ID != "msg-1" {
			t.Fatalf("unexpected message: %+v", m)
		}
		called = append(called, "h1")
	})
	reg.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		if s != fakeSession {
			t.Fatalf("expected wrapper session to be passed through")
		}
		called = append(called, "h2")
	})

	adder := &testDiscordgoAdder{}
	reg.RegisterWithSession(adder, fakeSession)
	if adder.handler == nil {
		t.Fatalf("expected a discordgo MessageCreate handler to be registered")
	}

	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:     "msg-1",
		Author: &discordgo.User{ID: "user-1"},
	}})

	if len(called) != 2 || called[0] != "h1" || called[1] != "h2" {
		t.Fatalf("unexpected call order: %v", called)
	}
}

func TestMessageRegistry_RegisterWithSession_RecoversHandlerPanic(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	reg := NewMessageRegistry(slog.Default())

	called := false
	reg.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		panic("boom")
	})
	reg.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		called = true
	})

	adder := &testDiscordgoAdder{}
	reg.RegisterWithSession(adder, fakeSession)
	if adder.handler == nil {
		t.Fatalf("expected a discordgo MessageCreate handler to be registered")
	}

	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:      "msg-1",
			Author:  &discordgo.User{ID: "user-1"},
			Content: "hello",
		},
	})

	if !called {
		t.Fatalf("expected subsequent handler to run after panic recovery")
	}
}

func TestMessageRegistry_RegisterWithSession_NilMessagePayloadIgnored(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	reg := NewMessageRegistry(slog.Default())

	called := false
	reg.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		called = true
	})

	adder := &testDiscordgoAdder{}
	reg.RegisterWithSession(adder, fakeSession)
	if adder.handler == nil {
		t.Fatalf("expected a discordgo MessageCreate handler to be registered")
	}

	adder.handler(&discordgo.Session{}, nil)
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{})
	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{ID: "m", ChannelID: "c"}})

	if called {
		t.Fatalf("expected handlers not to run for nil/invalid payloads")
	}
}
