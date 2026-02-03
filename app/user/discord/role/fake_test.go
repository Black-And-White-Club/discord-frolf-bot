package role

import (
	"context"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// FakeEventBus is a programmable fake for eventbus.EventBus
type FakeEventBus struct {
	PublishFunc           func(topic string, messages ...*message.Message) error
	SubscribeFunc         func(ctx context.Context, topic string) (<-chan *message.Message, error)
	CloseFunc             func() error
	GetNATSConnectionFunc func() *nats.Conn
	GetJetStreamFunc      func() jetstream.JetStream
	GetHealthCheckersFunc func() []eventbus.HealthChecker
	CreateStreamFunc      func(ctx context.Context, streamName string) error
	SubscribeForTestFunc  func(ctx context.Context, topic string) (<-chan *message.Message, error)
}

func (f *FakeEventBus) Publish(topic string, messages ...*message.Message) error {
	if f.PublishFunc != nil {
		return f.PublishFunc(topic, messages...)
	}
	return nil
}

func (f *FakeEventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if f.SubscribeFunc != nil {
		return f.SubscribeFunc(ctx, topic)
	}
	return nil, nil
}

func (f *FakeEventBus) Close() error {
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}
	return nil
}

func (f *FakeEventBus) GetNATSConnection() *nats.Conn {
	if f.GetNATSConnectionFunc != nil {
		return f.GetNATSConnectionFunc()
	}
	return nil
}

func (f *FakeEventBus) GetJetStream() jetstream.JetStream {
	if f.GetJetStreamFunc != nil {
		return f.GetJetStreamFunc()
	}
	return nil
}

func (f *FakeEventBus) GetHealthCheckers() []eventbus.HealthChecker {
	if f.GetHealthCheckersFunc != nil {
		return f.GetHealthCheckersFunc()
	}
	return nil
}

func (f *FakeEventBus) CreateStream(ctx context.Context, streamName string) error {
	if f.CreateStreamFunc != nil {
		return f.CreateStreamFunc(ctx, streamName)
	}
	return nil
}

func (f *FakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if f.SubscribeForTestFunc != nil {
		return f.SubscribeForTestFunc(ctx, topic)
	}
	return nil, nil
}

// Ensure interface compliance
var _ eventbus.EventBus = (*FakeEventBus)(nil)

// FakeISInterface is a programmable fake for storage.ISInterface
type FakeISInterface[T any] struct {
	*storage.FakeStorage[T]
}

func NewFakeISInterface[T any]() *FakeISInterface[T] {
	return &FakeISInterface[T]{
		FakeStorage: storage.NewFakeStorage[T](),
	}
}

func (f *FakeISInterface[T]) RecordFunc(method string) {
	f.FakeStorage.RecordCall(method)
}

// Ensure interface compliance
var _ storage.ISInterface[any] = (*FakeISInterface[any])(nil)

// FakeHelpers is a programmable fake for utils.Helpers
type FakeHelpers struct {
	CreateNewMessageFunc    func(payload interface{}, topic string) (*message.Message, error)
	CreateResultMessageFunc func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error)
	UnmarshalPayloadFunc    func(msg *message.Message, payload interface{}) error
}

func (f *FakeHelpers) CreateNewMessage(payload interface{}, topic string) (*message.Message, error) {
	if f.CreateNewMessageFunc != nil {
		return f.CreateNewMessageFunc(payload, topic)
	}
	return &message.Message{}, nil
}

func (f *FakeHelpers) CreateResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	if f.CreateResultMessageFunc != nil {
		return f.CreateResultMessageFunc(originalMsg, payload, topic)
	}
	return &message.Message{}, nil
}

func (f *FakeHelpers) UnmarshalPayload(msg *message.Message, payload interface{}) error {
	if f.UnmarshalPayloadFunc != nil {
		return f.UnmarshalPayloadFunc(msg, payload)
	}
	return nil
}

// FakeDiscordMetrics is a programmable fake for discordmetrics.DiscordMetrics
type FakeDiscordMetrics struct {
	RecordAPIRequestDurationFunc  func(ctx context.Context, endpoint string, duration time.Duration)
	RecordAPIRequestFunc          func(ctx context.Context, endpoint string)
	RecordAPIErrorFunc            func(ctx context.Context, endpoint string, errorType string)
	RecordRateLimitFunc           func(ctx context.Context, endpoint string, resetTime time.Duration)
	RecordWebsocketEventFunc      func(ctx context.Context, eventType string)
	RecordWebsocketReconnectFunc  func(ctx context.Context)
	RecordWebsocketDisconnectFunc func(ctx context.Context, reason string)
	RecordHandlerAttemptFunc      func(ctx context.Context, handlerName string)
	RecordHandlerSuccessFunc      func(ctx context.Context, handlerName string)
	RecordHandlerFailureFunc      func(ctx context.Context, handlerName string)
	RecordHandlerDurationFunc     func(ctx context.Context, handlerName string, duration time.Duration)
}

func (f *FakeDiscordMetrics) RecordAPIRequestDuration(ctx context.Context, endpoint string, duration time.Duration) {
	if f.RecordAPIRequestDurationFunc != nil {
		f.RecordAPIRequestDurationFunc(ctx, endpoint, duration)
	}
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

func (f *FakeDiscordMetrics) RecordRateLimit(ctx context.Context, endpoint string, resetTime time.Duration) {
	if f.RecordRateLimitFunc != nil {
		f.RecordRateLimitFunc(ctx, endpoint, resetTime)
	}
}

func (f *FakeDiscordMetrics) RecordWebsocketEvent(ctx context.Context, eventType string) {
	if f.RecordWebsocketEventFunc != nil {
		f.RecordWebsocketEventFunc(ctx, eventType)
	}
}

func (f *FakeDiscordMetrics) RecordWebsocketReconnect(ctx context.Context) {
	if f.RecordWebsocketReconnectFunc != nil {
		f.RecordWebsocketReconnectFunc(ctx)
	}
}

func (f *FakeDiscordMetrics) RecordWebsocketDisconnect(ctx context.Context, reason string) {
	if f.RecordWebsocketDisconnectFunc != nil {
		f.RecordWebsocketDisconnectFunc(ctx, reason)
	}
}

func (f *FakeDiscordMetrics) RecordHandlerAttempt(ctx context.Context, handlerName string) {
	if f.RecordHandlerAttemptFunc != nil {
		f.RecordHandlerAttemptFunc(ctx, handlerName)
	}
}

func (f *FakeDiscordMetrics) RecordHandlerSuccess(ctx context.Context, handlerName string) {
	if f.RecordHandlerSuccessFunc != nil {
		f.RecordHandlerSuccessFunc(ctx, handlerName)
	}
}

func (f *FakeDiscordMetrics) RecordHandlerFailure(ctx context.Context, handlerName string) {
	if f.RecordHandlerFailureFunc != nil {
		f.RecordHandlerFailureFunc(ctx, handlerName)
	}
}

func (f *FakeDiscordMetrics) RecordHandlerDuration(ctx context.Context, handlerName string, duration time.Duration) {
	if f.RecordHandlerDurationFunc != nil {
		f.RecordHandlerDurationFunc(ctx, handlerName, duration)
	}
}
