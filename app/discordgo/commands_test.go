package discord

import (
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRegisterCommands_Idempotent_SkipsExistingGuildCommands(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	// Configure the fake to return existing commands
	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	// Return some existing commands - the ones not listed here should be created
	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{
			{Name: "updaterole"},
			{Name: "createround"},
			nil,
			{Name: ""},
		}, nil
	}

	var created []string
	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		created = append(created, cmd.Name)
		return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the expected commands were created (claimtag, set-udisc-name, dashboard)
	want := map[string]bool{"claimtag": true, "set-udisc-name": true, "dashboard": true}
	if len(created) != 3 {
		t.Fatalf("expected 3 created commands, got %d: %v", len(created), created)
	}
	for _, name := range created {
		if !want[name] {
			t.Fatalf("unexpected created command: %q (all created: %v)", name, created)
		}
	}

	// Verify trace includes expected calls
	trace := fs.Trace()
	if !slices.Contains(trace, "GetBotUser") {
		t.Error("expected GetBotUser to be called")
	}
	if !slices.Contains(trace, "ApplicationCommands") {
		t.Error("expected ApplicationCommands to be called")
	}
	if !slices.Contains(trace, "ApplicationCommandCreate") {
		t.Error("expected ApplicationCommandCreate to be called")
	}
}

func TestRegisterCommands_Idempotent_AllCommandsPresent_NoCreates(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	// All 5 commands already exist
	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{
			{Name: "updaterole"},
			{Name: "createround"},
			{Name: "claimtag"},
			{Name: "set-udisc-name"},
			{Name: "dashboard"},
		}, nil
	}

	var createCalled bool
	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		createCalled = true
		return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCalled {
		t.Error("expected ApplicationCommandCreate to not be called when all commands are present")
	}
}

func TestRegisterCommands_ListError_FallsBackToCreate(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	// Simulate an error when listing commands
	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return nil, errors.New("list failed")
	}

	var createCount int
	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		createCount++
		return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created all 5 commands since list failed
	if createCount != 5 {
		t.Fatalf("expected 5 commands to be created (fallback behavior), got %d", createCount)
	}
}
