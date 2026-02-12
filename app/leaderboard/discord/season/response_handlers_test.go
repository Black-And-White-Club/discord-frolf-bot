package season

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel"
)

func TestSeasonManager_HandleSeasonStarted(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	logger := testutils.NoOpLogger()
	fakeMetrics := &testutils.FakeDiscordMetrics{}

	manager := NewSeasonManager(
		fakeSession,
		nil,
		logger,
		nil,
		nil,
		nil,
		nil,
		fakeGuildConfigCache, // Use fake cache
		otel.Tracer("test"),
		fakeMetrics,
	)

	// Mock GuildConfigCache to return a valid channel ID
	fakeGuildConfigCache.GetFunc = func(ctx context.Context, key string) (storage.GuildConfig, error) {
		return storage.GuildConfig{LeaderboardChannelID: "leaderboard-channel"}, nil
	}

	// Mock ChannelMessageSend
	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if channelID != "leaderboard-channel" {
			t.Errorf("expected channel ID leaderboard-channel, got %s", channelID)
		}
		if !strings.Contains(content, "New Season Started") {
			t.Errorf("expected message to contain 'New Season Started', got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.StartNewSeasonSuccessPayloadV1{
		SeasonID:   "season-1",
		SeasonName: "Test Season",
		GuildID:    "guild-1",
	}

	manager.HandleSeasonStarted(context.Background(), payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}

func TestSeasonManager_HandleSeasonStarted_NoChannel(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	logger := testutils.NoOpLogger()
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	manager := NewSeasonManager(
		fakeSession,
		nil,
		logger,
		nil,
		nil,
		fakeResolver,
		nil,
		fakeGuildConfigCache,
		otel.Tracer("test"),
		fakeMetrics,
	)

	// Mock Cache Miss
	fakeGuildConfigCache.GetFunc = func(ctx context.Context, key string) (storage.GuildConfig, error) {
		return storage.GuildConfig{}, errors.New("not found")
	}
	
	// Mock Resolver Miss (returns empty config or error)
	fakeResolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
		return &storage.GuildConfig{}, nil
	}

	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		t.Error("expected ChannelMessageSend NOT to be called")
		return nil, nil
	}

	payload := &leaderboardevents.StartNewSeasonSuccessPayloadV1{
		SeasonID:   "season-1",
		SeasonName: "Test Season",
		GuildID:    "guild-1",
	}

	manager.HandleSeasonStarted(context.Background(), payload)
}

func TestSeasonManager_HandleSeasonStandings(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	logger := testutils.NoOpLogger()
	fakeMetrics := &testutils.FakeDiscordMetrics{}

	manager := NewSeasonManager(
		fakeSession,
		nil,
		logger,
		nil,
		nil,
		nil,
		nil,
		fakeGuildConfigCache,
		otel.Tracer("test"),
		fakeMetrics,
	)

	fakeGuildConfigCache.GetFunc = func(ctx context.Context, key string) (storage.GuildConfig, error) {
		return storage.GuildConfig{LeaderboardChannelID: "leaderboard-channel"}, nil
	}

	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if !strings.Contains(content, "<@user-1>") {
			t.Errorf("expected message to contain '<@user-1>', got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.GetSeasonStandingsResponsePayloadV1{
		SeasonID: "season-1",
		GuildID:  sharedtypes.GuildID("guild-1"),
		Standings: []leaderboardevents.SeasonStandingItemV1{
			{MemberID: "user-1", TotalPoints: 100, RoundsPlayed: 5}, // We'll mock username resolution or just check ID in string if username fetch isn't happening here (it's not, it uses <@ID>)
		},
	}
	// The code uses <@MemberID> so we don't need to mock User resolution in Session.

	manager.HandleSeasonStandings(context.Background(), payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}

func TestSeasonManager_HandleSeasonStartFailed(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	logger := testutils.NoOpLogger()
	fakeMetrics := &testutils.FakeDiscordMetrics{}

	manager := NewSeasonManager(
		fakeSession,
		nil,
		logger,
		nil,
		nil,
		nil,
		nil,
		fakeGuildConfigCache,
		otel.Tracer("test"),
		fakeMetrics,
	)

	fakeGuildConfigCache.GetFunc = func(ctx context.Context, key string) (storage.GuildConfig, error) {
		return storage.GuildConfig{LeaderboardChannelID: "leaderboard-channel"}, nil
	}

	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if !strings.Contains(content, "Failed to start season") {
			t.Errorf("expected message to contain failure text, got %s", content)
		}
		if !strings.Contains(content, "something went wrong") {
			t.Errorf("expected message to contain reason, got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.AdminFailedPayloadV1{
		GuildID: sharedtypes.GuildID("guild-1"),
		Reason:  "something went wrong",
	}

	manager.HandleSeasonStartFailed(context.Background(), payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}
