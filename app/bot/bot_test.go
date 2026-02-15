package bot

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bot.syncGuildCommands(ctx, []*discordgo.Guild{{ID: "g1"}})
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

func TestSyncGuildCommands_SkipsAlreadySyncedManifest(t *testing.T) {
	resolver := &testutils.FakeGuildConfigResolver{}
	resolver.IsGuildSetupCompleteFunc = func(guildID string) bool {
		return guildID == "g1"
	}

	calls := 0
	bot := &DiscordBot{
		Logger:                 testLogger(),
		GuildConfigResolver:    resolver,
		commandSyncDelay:       0,
		commandSyncWorkers:     1,
		commandManifestVersion: "v-test",
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, guildID string) error {
			calls++
			return nil
		},
	}

	guilds := []*discordgo.Guild{{ID: "g1"}}
	bot.syncGuildCommands(context.Background(), guilds)
	bot.syncGuildCommands(context.Background(), guilds)

	if calls != 1 {
		t.Fatalf("expected registrar to run once for current manifest, got %d", calls)
	}
}

func TestSyncGuildCommands_RetriesPreviouslySkippedGuild(t *testing.T) {
	var setupChecks atomic.Int32
	resolver := &testutils.FakeGuildConfigResolver{}
	resolver.IsGuildSetupCompleteFunc = func(guildID string) bool {
		return setupChecks.Add(1) >= 2
	}

	synced := make(chan struct{}, 1)
	bot := &DiscordBot{
		Logger:                 testLogger(),
		GuildConfigResolver:    resolver,
		commandSyncDelay:       0,
		commandSyncWorkers:     1,
		commandSyncRetryDelay:  10 * time.Millisecond,
		commandManifestVersion: "v-test",
		commandRegistrar: func(_ discord.Session, _ *slog.Logger, guildID string) error {
			select {
			case synced <- struct{}{}:
			default:
			}
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	bot.syncGuildCommands(ctx, []*discordgo.Guild{{ID: "g1"}})

	select {
	case <-synced:
	case <-ctx.Done():
		t.Fatal("expected retry sync to register commands after setup completes")
	}
}

func TestNewDiscordBot_ReturnsResolverInitializationError(t *testing.T) {
	t.Cleanup(func() {
		newEventBusFactory = eventbus.NewEventBus
		newGuildConfigResolverFactory = guildconfig.NewResolver
	})

	newEventBusFactory = func(
		ctx context.Context,
		natsURL string,
		logger *slog.Logger,
		serviceName string,
		metrics eventbusmetrics.EventBusMetrics,
		tracer trace.Tracer,
	) (eventbus.EventBus, error) {
		return &testutils.FakeEventBus{}, nil
	}

	wantErr := errors.New("resolver init failed")
	newGuildConfigResolverFactory = func(
		ctx context.Context,
		eventBus eventbus.EventBus,
		cache storage.ISInterface[storage.GuildConfig],
		cfg *guildconfig.ResolverConfig,
	) (*guildconfig.Resolver, error) {
		return nil, wantErr
	}

	cfg := &config.Config{}
	stores := storage.NewStores(context.Background())
	bot, err := NewDiscordBot(
		discord.NewFakeSession(),
		cfg,
		testLogger(),
		stores,
		&testutils.FakeDiscordMetrics{},
		eventbusmetrics.NewNoop(),
		trace.NewNoopTracerProvider().Tracer("test"),
		utils.NewHelper(testLogger()),
	)
	if err == nil {
		t.Fatalf("expected constructor error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped resolver error, got %v", err)
	}
	if bot != nil {
		t.Fatalf("expected nil bot when constructor fails")
	}
}

func TestRegisterGatewayLifecycleHandlers_EmitsMetrics(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	var (
		connectHandler    func(*discordgo.Session, *discordgo.Connect)
		disconnectHandler func(*discordgo.Session, *discordgo.Disconnect)
		resumedHandler    func(*discordgo.Session, *discordgo.Resumed)
	)
	fakeSession.AddHandlerFunc = func(handler interface{}) func() {
		switch h := handler.(type) {
		case func(*discordgo.Session, *discordgo.Connect):
			connectHandler = h
		case func(*discordgo.Session, *discordgo.Disconnect):
			disconnectHandler = h
		case func(*discordgo.Session, *discordgo.Resumed):
			resumedHandler = h
		}
		return func() {}
	}

	var eventTypes []string
	var reconnects int
	var disconnectReasons []string
	metrics := &testutils.FakeDiscordMetrics{
		RecordWebsocketEventFunc: func(ctx context.Context, eventType string) {
			eventTypes = append(eventTypes, eventType)
		},
		RecordWebsocketReconnectFunc: func(ctx context.Context) {
			reconnects++
		},
		RecordWebsocketDisconnectFunc: func(ctx context.Context, reason string) {
			disconnectReasons = append(disconnectReasons, reason)
		},
	}

	bot := &DiscordBot{
		Session: fakeSession,
		Logger:  testLogger(),
		Metrics: metrics,
	}
	bot.setGatewayContext("session-123", 7)
	bot.registerGatewayLifecycleHandlers()

	if connectHandler == nil || disconnectHandler == nil || resumedHandler == nil {
		t.Fatalf("expected connect/disconnect/resumed handlers to be registered")
	}

	connectHandler(&discordgo.Session{}, &discordgo.Connect{})
	connectHandler(&discordgo.Session{}, &discordgo.Connect{})
	disconnectHandler(&discordgo.Session{}, &discordgo.Disconnect{})
	resumedHandler(&discordgo.Session{}, &discordgo.Resumed{})

	if len(eventTypes) != 3 || eventTypes[0] != "connect" || eventTypes[1] != "connect" || eventTypes[2] != "resumed" {
		t.Fatalf("unexpected websocket event sequence: %#v", eventTypes)
	}
	if reconnects != 2 {
		t.Fatalf("expected 2 reconnect metric emissions, got %d", reconnects)
	}
	if len(disconnectReasons) != 1 || disconnectReasons[0] != "gateway_disconnect" {
		t.Fatalf("unexpected disconnect reasons: %#v", disconnectReasons)
	}
}
