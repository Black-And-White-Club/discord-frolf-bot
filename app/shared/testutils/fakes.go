package testutils

import (
	"context"
	"time"

	"io"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func NoOpLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// FakeEventBus is a programmable fake for EventBus
type FakeEventBus struct {
	PublishFunc func(topic string, messages ...*message.Message) error
}

func (f *FakeEventBus) Publish(topic string, messages ...*message.Message) error {
	if f.PublishFunc != nil {
		return f.PublishFunc(topic, messages...)
	}
	return nil
}

func (f *FakeEventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return nil, nil
}

func (f *FakeEventBus) Close() error {
	return nil
}

func (f *FakeEventBus) GetNATSConnection() *nc.Conn {
	return nil
}

func (f *FakeEventBus) GetJetStream() jetstream.JetStream {
	return nil
}

func (f *FakeEventBus) GetHealthCheckers() []eventbus.HealthChecker {
	return nil
}

func (f *FakeEventBus) CreateStream(ctx context.Context, streamName string) error {
	return nil
}

func (f *FakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return nil, nil
}

// FakeHelpers is a programmable fake for Helpers
type FakeHelpers struct {
	CreateResultMessageFunc func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error)
	CreateNewMessageFunc    func(payload interface{}, topic string) (*message.Message, error)
	UnmarshalPayloadFunc    func(msg *message.Message, payload interface{}) error
}

func (f *FakeHelpers) CreateResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	if f.CreateResultMessageFunc != nil {
		return f.CreateResultMessageFunc(originalMsg, payload, topic)
	}
	return nil, nil
}

func (f *FakeHelpers) CreateNewMessage(payload interface{}, topic string) (*message.Message, error) {
	if f.CreateNewMessageFunc != nil {
		return f.CreateNewMessageFunc(payload, topic)
	}
	return nil, nil
}

func (f *FakeHelpers) UnmarshalPayload(msg *message.Message, payload interface{}) error {
	if f.UnmarshalPayloadFunc != nil {
		return f.UnmarshalPayloadFunc(msg, payload)
	}
	return nil
}

// FakeStorage is a generic programmable fake for storage interfaces
type FakeStorage[T any] struct {
	GetFunc    func(ctx context.Context, key string) (T, error)
	SetFunc    func(ctx context.Context, key string, value T) error
	DeleteFunc func(ctx context.Context, key string)
	ListFunc   func(ctx context.Context) ([]T, error)
}

func (f *FakeStorage[T]) Get(ctx context.Context, key string) (T, error) {
	if f.GetFunc != nil {
		return f.GetFunc(ctx, key)
	}
	var zero T
	return zero, nil
}

func (f *FakeStorage[T]) Set(ctx context.Context, key string, value T) error {
	if f.SetFunc != nil {
		return f.SetFunc(ctx, key, value)
	}
	return nil
}

func (f *FakeStorage[T]) Delete(ctx context.Context, key string) {
	if f.DeleteFunc != nil {
		f.DeleteFunc(ctx, key)
	}
}

func (f *FakeStorage[T]) List(ctx context.Context) ([]T, error) {
	if f.ListFunc != nil {
		return f.ListFunc(ctx)
	}
	return nil, nil
}

// FakeGuildConfigResolver is a programmable fake for GuildConfigResolver
type FakeGuildConfigResolver struct {
	GetGuildConfigFunc       func(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	IsGuildSetupCompleteFunc func(guildID string) bool
}

func (f *FakeGuildConfigResolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	if f.GetGuildConfigFunc != nil {
		return f.GetGuildConfigFunc(ctx, guildID)
	}
	return &storage.GuildConfig{}, nil
}

func (f *FakeGuildConfigResolver) RequestGuildConfigAsync(ctx context.Context, guildID string) {
}

func (f *FakeGuildConfigResolver) IsGuildSetupComplete(guildID string) bool {
	if f.IsGuildSetupCompleteFunc != nil {
		return f.IsGuildSetupCompleteFunc(guildID)
	}
	return true
}

func (f *FakeGuildConfigResolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
}

func (f *FakeGuildConfigResolver) HandleBackendError(ctx context.Context, guildID string, err error) {
}

func (f *FakeGuildConfigResolver) ClearInflightRequest(ctx context.Context, guildID string) {
}

// FakeDiscordMetrics is a programmable fake for DiscordMetrics
type FakeDiscordMetrics struct {
	RecordAPIRequestFunc         func(ctx context.Context, operation string)
	RecordAPIErrorFunc           func(ctx context.Context, operation, errorType string)
	RecordAPIRequestDurationFunc func(ctx context.Context, operation string, duration time.Duration)
}

func (f *FakeDiscordMetrics) RecordAPIRequest(ctx context.Context, endpoint string) {
	if f.RecordAPIRequestFunc != nil {
		f.RecordAPIRequestFunc(ctx, endpoint)
	}
}

func (f *FakeDiscordMetrics) RecordAPIError(ctx context.Context, endpoint string, errorType string) {
	if f.RecordAPIErrorFunc != nil {
		f.RecordAPIErrorFunc(ctx, endpoint, errorType)
	}
}

func (f *FakeDiscordMetrics) RecordAPIRequestDuration(ctx context.Context, endpoint string, duration time.Duration) {
	if f.RecordAPIRequestDurationFunc != nil {
		f.RecordAPIRequestDurationFunc(ctx, endpoint, duration)
	}
}

func (f *FakeDiscordMetrics) RecordRateLimit(ctx context.Context, endpoint string, resetTime time.Duration) {
}

func (f *FakeDiscordMetrics) RecordWebsocketEvent(ctx context.Context, eventType string) {
}

func (f *FakeDiscordMetrics) RecordWebsocketReconnect(ctx context.Context) {
}

func (f *FakeDiscordMetrics) RecordWebsocketDisconnect(ctx context.Context, reason string) {
}

func (f *FakeDiscordMetrics) RecordHandlerAttempt(ctx context.Context, handlerName string) {
}

func (f *FakeDiscordMetrics) RecordHandlerSuccess(ctx context.Context, handlerName string) {
}

func (f *FakeDiscordMetrics) RecordHandlerFailure(ctx context.Context, handlerName string) {
}

func (f *FakeDiscordMetrics) RecordHandlerDuration(ctx context.Context, handlerName string, duration time.Duration) {
}

// Interface assertions
var _ eventbus.EventBus = (*FakeEventBus)(nil)
var _ utils.Helpers = (*FakeHelpers)(nil)
var _ storage.ISInterface[any] = (*FakeStorage[any])(nil)
var _ guildconfig.GuildConfigResolver = (*FakeGuildConfigResolver)(nil)
var _ discordmetrics.DiscordMetrics = (*FakeDiscordMetrics)(nil)
