package interactions

import (
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
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
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	wrapperSession := discordmocks.NewMockSession(ctrl)
	wrapper := discord.Session(wrapperSession)
	reg := NewMessageRegistry()

	var called []string
	reg.RegisterMessageCreateHandler(func(s discord.Session, m *discordgo.MessageCreate) {
		if s != wrapper {
			t.Fatalf("expected wrapper session to be passed through")
		}
		if m == nil || m.Message == nil || m.Message.ID != "msg-1" {
			t.Fatalf("unexpected message: %+v", m)
		}
		called = append(called, "h1")
	})
	reg.RegisterMessageCreateHandler(func(s discord.Session, m *discordgo.MessageCreate) {
		if s != wrapper {
			t.Fatalf("expected wrapper session to be passed through")
		}
		called = append(called, "h2")
	})

	adder := &testDiscordgoAdder{}
	reg.RegisterWithSession(adder, wrapper)
	if adder.handler == nil {
		t.Fatalf("expected a discordgo MessageCreate handler to be registered")
	}

	adder.handler(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{ID: "msg-1"}})

	if len(called) != 2 || called[0] != "h1" || called[1] != "h2" {
		t.Fatalf("unexpected call order: %v", called)
	}
}
