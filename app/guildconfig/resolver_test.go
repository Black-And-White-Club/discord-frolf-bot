package guildconfig

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// fakeEventBus is a minimal test double capturing publish calls and allowing injection of publish errors.
type fakeEventBus struct {
	mu         sync.Mutex
	published  []string
	publishErr error
}

func (f *fakeEventBus) Publish(topic string, messages ...*message.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, topic)
	return f.publishErr
}

func (f *fakeEventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return nil, errors.New("not implemented in fake")
}
func (f *fakeEventBus) Close() error                                              { return nil }
func (f *fakeEventBus) GetNATSConnection() *nc.Conn                               { return nil }
func (f *fakeEventBus) GetJetStream() jetstream.JetStream                         { return nil }
func (f *fakeEventBus) GetHealthCheckers() []eventbus.HealthChecker               { return nil }
func (f *fakeEventBus) CreateStream(ctx context.Context, streamName string) error { return nil }

// SubscribeForTest implements eventbus.EventBus.SubscribeForTest used by tests.
// It returns a channel that the test harness can read from if needed.
func (f *fakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message, 1)
	return ch, nil
}

func TestNewResolver_ReturnsErrorOnInvalidConfig(t *testing.T) {
	fb := &fakeEventBus{}
	bad := &ResolverConfig{RequestTimeout: 0, ResponseTimeout: time.Second}
	_, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), bad)
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestNewResolver_OK(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.config != cfg {
		t.Fatalf("resolver did not keep provided config")
	}
}

func TestNewResolverWithDefaults(t *testing.T) {
	fb := &fakeEventBus{}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), DefaultResolverConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	def := DefaultResolverConfig()
	if r.config.RequestTimeout != def.RequestTimeout || r.config.ResponseTimeout != def.ResponseTimeout {
		t.Fatalf("defaults mismatch: got %+v want %+v", r.config, def)
	}
}

func TestResolver_GetGuildConfig_DelegatesToContextVersion(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 5 * time.Millisecond * 2}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Force a timeout path: no backend response -> expect ConfigLoadingError after response timeout
	_, err = r.GetGuildConfig("g1")
	if err == nil {
		t.Fatalf("expected error due to timeout/loading")
	}
}

func TestResolver_GetGuildConfigWithContext_Scenarios(t *testing.T) {
	type scenario struct {
		name          string
		publishErr    error
		respond       func(r *Resolver, guildID string)
		expectErrType interface{}
		expectConfig  *storage.GuildConfig
	}
	guildID := "guild123"
	baseConfig := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 20 * time.Millisecond}
	cases := []scenario{
		{
			name: "success",
			respond: func(r *Resolver, gid string) {
				time.AfterFunc(2*time.Millisecond, func() {
					r.HandleGuildConfigReceived(context.Background(), gid, &storage.GuildConfig{GuildID: gid, SignupChannelID: "c", EventChannelID: "e", LeaderboardChannelID: "l", RegisteredRoleID: "r"})
				})
			},
			expectConfig: &storage.GuildConfig{GuildID: guildID, SignupChannelID: "c", EventChannelID: "e", LeaderboardChannelID: "l", RegisteredRoleID: "r"},
		},
		{
			name:          "publish failure",
			publishErr:    errors.New("nats down"),
			expectErrType: &ConfigTemporaryError{},
		},
		{
			name: "permanent failure",
			respond: func(r *Resolver, gid string) {
				time.AfterFunc(2*time.Millisecond, func() {
					r.HandleGuildConfigRetrievalFailed(context.Background(), gid, "permanent failure: guild not configured", true)
				})
			},
			expectErrType: &ConfigNotFoundError{},
		},
		{
			name: "temporary failure",
			respond: func(r *Resolver, gid string) {
				time.AfterFunc(2*time.Millisecond, func() {
					r.HandleGuildConfigRetrievalFailed(context.Background(), gid, "temporary failure: connection timeout", false)
				})
			},
			expectErrType: &ConfigTemporaryError{},
		},
		{
			name:          "timeout loading",
			expectErrType: &ConfigLoadingError{},
		},
	}
	for _, sc := range cases {
		t.Run(sc.name, func(t *testing.T) {
			fb := &fakeEventBus{publishErr: sc.publishErr}
			r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), baseConfig)
			if err != nil {
				t.Fatalf("unexpected constructor error: %v", err)
			}
			if sc.respond != nil {
				sc.respond(r, guildID)
			}
			cfg, err := r.GetGuildConfigWithContext(context.Background(), guildID)
			if sc.expectErrType != nil {
				if err == nil {
					t.Fatalf("expected error of type %T", sc.expectErrType)
				}
				if reflect.TypeOf(err) != reflect.TypeOf(sc.expectErrType) {
					t.Fatalf("unexpected error type %T (val=%v) want %T", err, err, sc.expectErrType)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sc.expectConfig != nil {
				if !reflect.DeepEqual(cfg, sc.expectConfig) {
					t.Fatalf("config mismatch got %+v want %+v", cfg, sc.expectConfig)
				}
			}
		})
	}
}

func TestResolver_coordinateConfigRequest_PublishErrorSetsTemporaryError(t *testing.T) {
	fb := &fakeEventBus{publishErr: errors.New("publish fail")}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	req := &configRequest{ready: make(chan struct{})}
	r.inflightRequests.Store("g", req)
	r.coordinateConfigRequest(context.Background(), "g", req)
	<-req.ready
	if req.err == nil || reflect.TypeOf(req.err) != reflect.TypeOf(&ConfigTemporaryError{}) {
		t.Fatalf("expected temporary error got %v", req.err)
	}
}

func TestResolver_HandleGuildConfigReceived_CompletesRequest(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	req := &configRequest{ready: make(chan struct{})}
	r.inflightRequests.Store("g", req)
	gc := &storage.GuildConfig{GuildID: "g", SignupChannelID: "c", EventChannelID: "e", LeaderboardChannelID: "l", RegisteredRoleID: "r"}
	r.HandleGuildConfigReceived(context.Background(), "g", gc)
	<-req.ready
	if req.config != gc || req.err != nil {
		t.Fatalf("expected config set without error")
	}
}

func TestResolver_HandleGuildConfigRetrievalFailed_CompletesRequest(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	req := &configRequest{ready: make(chan struct{})}
	r.inflightRequests.Store("g", req)
	r.HandleGuildConfigRetrievalFailed(context.Background(), "g", "temporary failure: x", false)
	<-req.ready
	if req.err == nil || reflect.TypeOf(req.err) != reflect.TypeOf(&ConfigTemporaryError{}) {
		t.Fatalf("expected temporary failure error")
	}
}

func TestResolver_HandleBackendError_Classification(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	req := &configRequest{ready: make(chan struct{})}
	r.inflightRequests.Store("g", req)
	r.HandleBackendError(context.Background(), "g", errors.New("guild not configured"))
	<-req.ready
	if _, ok := req.err.(*ConfigNotFoundError); !ok {
		t.Fatalf("expected ConfigNotFoundError got %v", req.err)
	}
}

func TestResolver_IsGuildSetupComplete(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 15 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	// Success path: respond with config
	go func() {
		time.Sleep(2 * time.Millisecond)
		r.HandleGuildConfigReceived(context.Background(), "g", &storage.GuildConfig{GuildID: "g", SignupChannelID: "c", EventChannelID: "e", LeaderboardChannelID: "l", RegisteredRoleID: "r"})
	}()
	if !r.IsGuildSetupComplete("g") {
		t.Fatalf("expected setup complete true")
	}
	// Failure path: timeout leads to false
	if r.IsGuildSetupComplete("missing") {
		t.Fatalf("expected setup incomplete for missing guild")
	}
}

func TestResolver_recordErrorAndMetrics(t *testing.T) {
	fb := &fakeEventBus{}
	cfg := &ResolverConfig{RequestTimeout: 5 * time.Millisecond, ResponseTimeout: 10 * time.Millisecond}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	r.recordError(context.Background(), "g", "temporary")
	r.recordError(context.Background(), "g", "temporary")
	r.recordError(context.Background(), "g", "permanent")
	m := r.GetErrorMetrics()
	if m.TotalErrors != 3 || m.ErrorsByType["temporary"] != 2 || m.ErrorsByType["permanent"] != 1 || m.ErrorsByGuild["g"] != 3 {
		t.Fatalf("unexpected metrics %+v", m)
	}
	r.ResetErrorMetrics()
	m2 := r.GetErrorMetrics()
	if m2.TotalErrors != 0 || len(m2.ErrorsByType) != 0 || len(m2.ErrorsByGuild) != 0 {
		t.Fatalf("expected cleared metrics %+v", m2)
	}
}

// (GetErrorMetrics tested within TestResolver_recordErrorAndMetrics)

// (ResetErrorMetrics tested within TestResolver_recordErrorAndMetrics)

func TestResolver_LogConfigEvent_NoPanic(t *testing.T) {
	fb := &fakeEventBus{}
	r, err := NewResolver(context.Background(), fb, storage.NewInteractionStore[storage.GuildConfig](context.Background(), 1*time.Hour), DefaultResolverConfig())
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	r.LogConfigEvent(context.Background(), "test_event", "g", slog.String("k", "v"))
}

func Test_convertAttrsToAny(t *testing.T) {
	attrs := []slog.Attr{slog.String("a", "b"), slog.Int("n", 1)}
	got := convertAttrsToAny(attrs)
	if len(got) != len(attrs) {
		t.Fatalf("expected len %d got %d", len(attrs), len(got))
	}
	// Assert element types are slog.Attr
	if _, ok := got[0].(slog.Attr); !ok {
		t.Fatalf("expected first element to be slog.Attr, got %T", got[0])
	}
	if _, ok := got[1].(slog.Attr); !ok {
		t.Fatalf("expected second element to be slog.Attr, got %T", got[1])
	}
}
