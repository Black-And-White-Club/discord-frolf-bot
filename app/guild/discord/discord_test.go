package discord

import (
	"context"
	"io"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRegisterAllCommands_Global(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "").Return([]*discordgo.ApplicationCommand{}, nil)
	// Expect 2 global commands: frolf-setup and frolf-reset
	ms.EXPECT().ApplicationCommandCreate("bot", "", gomock.Any(), gomock.Any()).Times(2).Return(&discordgo.ApplicationCommand{ID: "cmd1"}, nil)

	gd := &GuildDiscord{session: ms, logger: logger}
	if err := gd.RegisterAllCommands(""); err != nil {
		t.Fatalf("RegisterAllCommands(global) unexpected error: %v", err)
	}
}

func TestRegisterAllCommands_Guild(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "g1").Return([]*discordgo.ApplicationCommand{}, nil)
	// Expect all guild commands to be created
	ms.EXPECT().ApplicationCommandCreate("bot", "g1", gomock.Any(), gomock.Any()).Times(4).Return(&discordgo.ApplicationCommand{ID: "c"}, nil)

	gd := &GuildDiscord{session: ms, logger: logger}
	if err := gd.RegisterAllCommands("g1"); err != nil {
		t.Fatalf("RegisterAllCommands(guild) unexpected error: %v", err)
	}
}

func TestUnregisterAllCommands(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()

	ms.EXPECT().GetBotUser().Return(&discordgo.User{ID: "bot"}, nil)
	ms.EXPECT().ApplicationCommands("bot", "g1", gomock.Any()).Return([]*discordgo.ApplicationCommand{
		{ID: "1", Name: "frolf-setup"},
		{ID: "2", Name: "updaterole"},
		{ID: "3", Name: "createround"},
	}, nil)
	// Expect deletes for non-setup commands only
	ms.EXPECT().ApplicationCommandDelete("bot", "g1", "2", gomock.Any()).Return(nil)
	ms.EXPECT().ApplicationCommandDelete("bot", "g1", "3", gomock.Any()).Return(nil)

	gd := &GuildDiscord{session: ms, logger: logger}
	if err := gd.UnregisterAllCommands("g1"); err != nil {
		t.Fatalf("UnregisterAllCommands unexpected error: %v", err)
	}
}

func TestNewGuildDiscord_And_GetSetupManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ms := discordmocks.NewMockSession(ctrl)
	logger := testLogger()
	helper := utils.NewHelper(logger)
	var resolver guildconfig.GuildConfigResolver = nil

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)

	gdIface, err := NewGuildDiscord(
		context.Background(),
		ms,
		mockEventBus, // eventbus
		logger,
		helper,
		nil, // config
		nil, // interactionStore
		nil, // tracer
		nil, // metrics
		resolver,
	)
	if err != nil {
		t.Fatalf("NewGuildDiscord error: %v", err)
	}
	gd := gdIface.(*GuildDiscord)
	if gd.GetSetupManager() == nil {
		t.Fatalf("expected non-nil setup manager")
	}
	if gd.GetResetManager() == nil {
		t.Fatalf("expected non-nil reset manager")
	}
}
