package discord

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestRegisterCommands_ReconcileGuildCommands_CreatesUpdatesDeletes(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return []*discordgo.ApplicationCommand{
			{
				ID:          "cmd-updaterole",
				Name:        "updaterole",
				Description: "Request a role for a user (Requires Editor role or higher)",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "The user to request a role for",
						Required:    true,
					},
				},
			},
			{
				ID:          "cmd-dashboard",
				Name:        "dashboard",
				Description: "old description",
			},
			{
				ID:          "cmd-deprecated",
				Name:        "legacy-command",
				Description: "deprecated",
			},
		}, nil
	}

	created := map[string]bool{}
	edited := map[string]bool{}
	deleted := map[string]bool{}

	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		created[cmd.Name] = true
		return &discordgo.ApplicationCommand{ID: "new-" + cmd.Name, Name: cmd.Name}, nil
	}
	fs.ApplicationCommandEditFunc = func(appID, guildID, cmdID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		edited[cmd.Name] = true
		return &discordgo.ApplicationCommand{ID: cmdID, Name: cmd.Name}, nil
	}
	fs.ApplicationCommandDeleteFunc = func(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
		deleted[cmdID] = true
		return nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"createround", "claimtag", "set-udisc-name", "season"} {
		if !created[name] {
			t.Fatalf("expected command %q to be created; created=%v", name, created)
		}
	}

	if !edited["dashboard"] {
		t.Fatalf("expected dashboard command to be edited; edited=%v", edited)
	}

	if !deleted["cmd-deprecated"] {
		t.Fatalf("expected legacy command to be deleted; deleted=%v", deleted)
	}
}

func TestRegisterCommands_AllCommandsCurrent_NoChanges(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		desired := desiredCommands("g1")
		current := make([]*discordgo.ApplicationCommand, 0, len(desired))
		for i, cmd := range desired {
			copyCmd := *cmd
			copyCmd.ID = "cmd-" + string(rune('a'+i))
			current = append(current, &copyCmd)
		}
		return current, nil
	}

	var createCalled, editCalled, deleteCalled bool
	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		createCalled = true
		return &discordgo.ApplicationCommand{ID: "new-" + cmd.Name, Name: cmd.Name}, nil
	}
	fs.ApplicationCommandEditFunc = func(appID, guildID, cmdID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		editCalled = true
		return &discordgo.ApplicationCommand{ID: cmdID, Name: cmd.Name}, nil
	}
	fs.ApplicationCommandDeleteFunc = func(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
		deleteCalled = true
		return nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCalled || editCalled || deleteCalled {
		t.Fatalf("expected no reconciliation writes, got create=%t edit=%t delete=%t", createCalled, editCalled, deleteCalled)
	}
}

func TestRegisterCommands_ListError_ReturnsError(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}
	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		return nil, errors.New("list failed")
	}

	if err := RegisterCommands(fs, logger, "g1"); err == nil {
		t.Fatal("expected list failure to be returned")
	}
}

func TestRegisterCommands_RetriesTransientCreateFailure(t *testing.T) {
	fs := NewFakeSession()
	logger := testLogger()

	fs.GetBotUserFunc = func() (*discordgo.User, error) {
		return &discordgo.User{ID: "bot"}, nil
	}

	fs.ApplicationCommandsFunc = func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
		desired := desiredCommands("g1")
		return []*discordgo.ApplicationCommand{
			{
				ID:          "cmd-updaterole",
				Name:        desired[0].Name,
				Description: desired[0].Description,
				Options:     desired[0].Options,
			},
			{
				ID:          "cmd-createround",
				Name:        desired[1].Name,
				Description: desired[1].Description,
				Options:     desired[1].Options,
			},
			{
				ID:          "cmd-claimtag",
				Name:        desired[2].Name,
				Description: desired[2].Description,
				Options:     desired[2].Options,
			},
			{
				ID:          "cmd-set-udisc-name",
				Name:        desired[3].Name,
				Description: desired[3].Description,
				Options:     desired[3].Options,
			},
			{
				ID:          "cmd-invite",
				Name:        desired[5].Name,
				Description: desired[5].Description,
				Options:     desired[5].Options,
			},
			{
				ID:                       "cmd-season",
				Name:                     desired[6].Name,
				Description:              desired[6].Description,
				Options:                  desired[6].Options,
				DefaultMemberPermissions: desired[6].DefaultMemberPermissions,
			},
		}, nil
	}

	createCalls := 0
	fs.ApplicationCommandCreateFunc = func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
		createCalls++
		if cmd.Name == "dashboard" && createCalls < 3 {
			return nil, &discordgo.RESTError{
				Response: &http.Response{StatusCode: http.StatusInternalServerError},
				Message:  &discordgo.APIErrorMessage{Code: 0, Message: "server error"},
			}
		}
		return &discordgo.ApplicationCommand{ID: cmd.Name + "-id"}, nil
	}

	if err := RegisterCommands(fs, logger, "g1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalls != 3 {
		t.Fatalf("expected 3 create attempts for transient failures, got %d", createCalls)
	}
}
