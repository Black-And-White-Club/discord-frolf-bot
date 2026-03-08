package discord

import (
	"context"
	"io"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRegisterAllCommands_Global(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	logger := testLogger()

	fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}
	fakeSession.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{}, nil
	}
	fakeSession.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		return &discordgo.ApplicationCommand{ID: "cmd1"}, nil
	}

	gd := &GuildDiscord{session: fakeSession, logger: logger}
	if err := gd.RegisterAllCommands(""); err != nil {
		t.Fatalf("RegisterAllCommands(global) unexpected error: %v", err)
	}
}

func TestRegisterAllCommands_Guild(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	logger := testLogger()

	fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}
	fakeSession.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{}, nil
	}
	fakeSession.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		return &discordgo.ApplicationCommand{ID: "c"}, nil
	}

	gd := &GuildDiscord{session: fakeSession, logger: logger}
	if err := gd.RegisterAllCommands("g1"); err != nil {
		t.Fatalf("RegisterAllCommands(guild) unexpected error: %v", err)
	}
}

func TestUnregisterAllCommands(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	logger := testLogger()

	fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}
	fakeSession.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{
			{ID: "1", Name: "frolf-setup"},
			{ID: "2", Name: "updaterole"},
			{ID: "3", Name: "createround"},
		}, nil
	}
	fakeSession.ApplicationCommandDeleteFunc = func(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
		return nil
	}

	gd := &GuildDiscord{session: fakeSession, logger: logger}
	if err := gd.UnregisterAllCommands("g1"); err != nil {
		t.Fatalf("UnregisterAllCommands unexpected error: %v", err)
	}
}

func TestNewGuildDiscord_And_GetSetupManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	logger := testLogger()
	helper := utils.NewHelper(logger)
	var resolver guildconfig.GuildConfigResolver = nil

	fakeEventBus := &testutils.FakeEventBus{}

	gdIface, err := NewGuildDiscord(
		context.Background(),
		fakeSession,
		fakeEventBus, // eventbus
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
