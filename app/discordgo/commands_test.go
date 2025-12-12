package discord

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRegisterCommands_Idempotent_SkipsExistingGuildCommands(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "g1").Return([]*discordgo.ApplicationCommand{
		{Name: "updaterole"},
		{Name: "createround"},
		nil,
		{Name: ""},
	}, nil)

	var created []string
	ms.EXPECT().ApplicationCommandCreate("bot", "g1", gomock.Any()).
		DoAndReturn(func(_ string, _ string, cmd *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
			created = append(created, cmd.Name)
			return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
		}).
		Times(2)

	if err := RegisterCommands(ms, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]bool{"claimtag": true, "set-udisc-name": true}
	if len(created) != 2 {
		t.Fatalf("expected 2 created commands, got %v", created)
	}
	for _, name := range created {
		if !want[name] {
			t.Fatalf("unexpected created command: %q (all created: %v)", name, created)
		}
	}
}

func TestRegisterCommands_Idempotent_AllCommandsPresent_NoCreates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "g1").Return([]*discordgo.ApplicationCommand{
		{Name: "updaterole"},
		{Name: "createround"},
		{Name: "claimtag"},
		{Name: "set-udisc-name"},
	}, nil)

	ms.EXPECT().ApplicationCommandCreate(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	if err := RegisterCommands(ms, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterCommands_ListError_FallsBackToCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "g1").Return(nil, errors.New("list failed"))

	ms.EXPECT().ApplicationCommandCreate("bot", "g1", gomock.Any()).
		DoAndReturn(func(_ string, _ string, cmd *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
			return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
		}).
		Times(4)

	if err := RegisterCommands(ms, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
