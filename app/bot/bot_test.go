package bot

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/bwmarrin/discordgo"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestSyncGuildCommands_EmptyGuildList_NoRegistrarCalls(t *testing.T) {
	bot := &DiscordBot{
		Logger:           testLogger(),
		commandSyncDelay: 0,
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, _ string) error {
			t.Fatalf("registrar should not be called for empty guild list")
			return nil
		},
	}

	bot.syncGuildCommands(context.Background(), nil)
	bot.syncGuildCommands(context.Background(), []*discordgo.Guild{})
}

func TestSyncGuildCommands_SkipsGuildsWithIncompleteSetup(t *testing.T) {
	resolver := &testutils.FakeGuildConfigResolver{}
	resolver.IsGuildSetupCompleteFunc = func(guildID string) bool {
		return guildID != "g1"
	}

	called := 0
	bot := &DiscordBot{
		Logger:              testLogger(),
		GuildConfigResolver: resolver,
		commandSyncDelay:    0,
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, _ string) error {
			called++
			return nil
		},
	}

	bot.syncGuildCommands(context.Background(), []*discordgo.Guild{{ID: "g1"}})
	if called != 0 {
		t.Fatalf("expected registrar not to be called, got %d", called)
	}
}

func TestSyncGuildCommands_RegistersSetupCompleteGuilds_ContinuesOnError(t *testing.T) {
	resolver := &testutils.FakeGuildConfigResolver{}
	resolver.IsGuildSetupCompleteFunc = func(guildID string) bool {
		return guildID == "g1" || guildID == "g2"
	}

	var got []string
	bot := &DiscordBot{
		Logger:              testLogger(),
		GuildConfigResolver: resolver,
		commandSyncDelay:    0,
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, guildID string) error {
			got = append(got, guildID)
			if guildID == "g1" {
				return errors.New("boom")
			}
			return nil
		},
	}

	bot.syncGuildCommands(context.Background(), []*discordgo.Guild{{ID: "g1"}, {ID: "g2"}})
	if len(got) != 2 || got[0] != "g1" || got[1] != "g2" {
		t.Fatalf("unexpected registrar calls: %v", got)
	}
}

func TestSyncGuildCommands_CanceledContext_StopsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bot := &DiscordBot{
		Logger:           testLogger(),
		commandSyncDelay: 0,
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, _ string) error {
			t.Fatalf("registrar should not be called when context is canceled")
			return nil
		},
	}

	bot.syncGuildCommands(ctx, []*discordgo.Guild{{ID: "g1"}})
}
