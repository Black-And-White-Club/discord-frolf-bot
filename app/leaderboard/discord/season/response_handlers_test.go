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
	fakeSession.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
		return &discordgo.Member{
			User: &discordgo.User{
				ID:       userID,
				Username: "fallback-user",
			},
			Nick: "Farr",
		}, nil
	}

	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if !strings.Contains(content, "**Farr** (<@839877196898238526>)") {
			t.Errorf("expected message to contain formatted member label, got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.GetSeasonStandingsResponsePayloadV1{
		SeasonID: "season-1",
		GuildID:  sharedtypes.GuildID("guild-1"),
		Standings: []leaderboardevents.SeasonStandingItemV1{
			{MemberID: "839877196898238526", TotalPoints: 100, RoundsPlayed: 5},
		},
	}

	manager.HandleSeasonStandings(context.Background(), payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}

func TestSeasonManager_HandleSeasonStandings_RawHandleFallback(t *testing.T) {
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
		if !strings.Contains(content, "@farrmich") {
			t.Errorf("expected message to contain raw handle, got %s", content)
		}
		if strings.Contains(content, "@@farrmich") {
			t.Errorf("expected no duplicated @ prefix, got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.GetSeasonStandingsResponsePayloadV1{
		SeasonID: "season-1",
		GuildID:  sharedtypes.GuildID("guild-1"),
		Standings: []leaderboardevents.SeasonStandingItemV1{
			{MemberID: "@farrmich", TotalPoints: 100, RoundsPlayed: 5},
		},
	}

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

func TestSeasonManager_HandleSeasonEnded_PrioritizesConfigChannel(t *testing.T) {
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

	// Mock GuildConfigCache to return a valid channel ID
	fakeGuildConfigCache.GetFunc = func(ctx context.Context, key string) (storage.GuildConfig, error) {
		return storage.GuildConfig{LeaderboardChannelID: "config-channel"}, nil
	}

	// Mock ChannelMessageSend
	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if channelID != "config-channel" {
			t.Errorf("expected channel ID config-channel, got %s", channelID)
		}
		if !strings.Contains(content, "Season Ended") {
			t.Errorf("expected message to contain 'Season Ended', got %s", content)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.EndSeasonSuccessPayloadV1{
		GuildID: sharedtypes.GuildID("guild-1"),
	}

	// Put a different channel ID in context to verify priority
	ctx := context.WithValue(context.Background(), channelIDKey, "context-channel")
	manager.HandleSeasonEnded(ctx, payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}

func TestSeasonManager_HandleSeasonEnded_FallbackToContext(t *testing.T) {
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
	// Mock Resolver Miss
	fakeResolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
		return &storage.GuildConfig{}, nil
	}

	// Mock ChannelMessageSend
	messageSent := false
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		messageSent = true
		if channelID != "context-channel" {
			t.Errorf("expected channel ID context-channel, got %s", channelID)
		}
		return &discordgo.Message{}, nil
	}

	payload := &leaderboardevents.EndSeasonSuccessPayloadV1{
		GuildID: sharedtypes.GuildID("guild-1"),
	}

	ctx := context.WithValue(context.Background(), channelIDKey, "context-channel")
	manager.HandleSeasonEnded(ctx, payload)

	if !messageSent {
		t.Error("expected ChannelMessageSend to be called")
	}
}
