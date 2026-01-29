package claimtag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
)

// simple in-memory ISInterface stub
type memStore struct{ m map[string]interface{} }

// Implement the new storage.ISInterface[T] signatures (context-aware, (T, error) returns)
func (s *memStore) Set(_ context.Context, id string, v interface{}) error {
	if s.m == nil {
		s.m = map[string]interface{}{}
	}
	s.m[id] = v
	return nil
}

func (s *memStore) Delete(_ context.Context, id string) {
	if s.m != nil {
		delete(s.m, id)
	}
}

func (s *memStore) Get(_ context.Context, id string) (interface{}, error) {
	if s.m == nil {
		return nil, errors.New("not found")
	}
	v, ok := s.m[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

// fake eventbus capturing publish
type fakeEventBus struct {
	topics []string
	err    error
}

func (f *fakeEventBus) Publish(topic string, _ ...*message.Message) error {
	f.topics = append(f.topics, topic)
	return f.err
}

func (f *fakeEventBus) Subscribe(context.Context, string) (<-chan *message.Message, error) {
	return nil, errors.New("not used")
}
func (f *fakeEventBus) Close() error                                { return nil }
func (f *fakeEventBus) GetNATSConnection() *nc.Conn                 { return nil }
func (f *fakeEventBus) GetJetStream() jetstream.JetStream           { return nil }
func (f *fakeEventBus) GetHealthCheckers() []eventbus.HealthChecker { return nil }
func (f *fakeEventBus) CreateStream(context.Context, string) error  { return nil }

// SubscribeForTest implements eventbus.EventBus.SubscribeForTest used by tests.
func (f *fakeEventBus) SubscribeForTest(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message, 1)
	return ch, nil
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeHelpers provides a programmable stub for utils.Helpers
type fakeHelpers struct {
	CreateNewMessageFunc    func(payload interface{}, topic string) (*message.Message, error)
	CreateResultMessageFunc func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error)
	UnmarshalPayloadFunc    func(msg *message.Message, payload interface{}) error
}

func (f *fakeHelpers) CreateNewMessage(payload interface{}, topic string) (*message.Message, error) {
	if f.CreateNewMessageFunc != nil {
		return f.CreateNewMessageFunc(payload, topic)
	}
	return &message.Message{}, nil
}

func (f *fakeHelpers) CreateResultMessage(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
	if f.CreateResultMessageFunc != nil {
		return f.CreateResultMessageFunc(originalMsg, payload, topic)
	}
	return &message.Message{}, nil
}

func (f *fakeHelpers) UnmarshalPayload(msg *message.Message, payload interface{}) error {
	if f.UnmarshalPayloadFunc != nil {
		return f.UnmarshalPayloadFunc(msg, payload)
	}
	return nil
}

// fakeGuildConfigResolver provides a programmable stub for GuildConfigResolver
type fakeGuildConfigResolver struct {
	GetGuildConfigWithContextFunc func(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	RequestGuildConfigAsyncFunc   func(ctx context.Context, guildID string)
	IsGuildSetupCompleteFunc      func(guildID string) bool
	HandleGuildConfigReceivedFunc func(ctx context.Context, guildID string, config *storage.GuildConfig)
	HandleBackendErrorFunc        func(ctx context.Context, guildID string, err error)
	ClearInflightRequestFunc      func(ctx context.Context, guildID string)
}

func (f *fakeGuildConfigResolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	if f.GetGuildConfigWithContextFunc != nil {
		return f.GetGuildConfigWithContextFunc(ctx, guildID)
	}
	return &storage.GuildConfig{}, nil
}

func (f *fakeGuildConfigResolver) RequestGuildConfigAsync(ctx context.Context, guildID string) {
	if f.RequestGuildConfigAsyncFunc != nil {
		f.RequestGuildConfigAsyncFunc(ctx, guildID)
	}
}

func (f *fakeGuildConfigResolver) IsGuildSetupComplete(guildID string) bool {
	if f.IsGuildSetupCompleteFunc != nil {
		return f.IsGuildSetupCompleteFunc(guildID)
	}
	return true
}

func (f *fakeGuildConfigResolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
	if f.HandleGuildConfigReceivedFunc != nil {
		f.HandleGuildConfigReceivedFunc(ctx, guildID, config)
	}
}

func (f *fakeGuildConfigResolver) HandleBackendError(ctx context.Context, guildID string, err error) {
	if f.HandleBackendErrorFunc != nil {
		f.HandleBackendErrorFunc(ctx, guildID, err)
	}
}

func (f *fakeGuildConfigResolver) ClearInflightRequest(ctx context.Context, guildID string) {
	if f.ClearInflightRequestFunc != nil {
		f.ClearInflightRequestFunc(ctx, guildID)
	}
}

func TestHandleClaimTagCommand_Variants(t *testing.T) {
	ctx := context.Background()
	tracer := otel.Tracer("test")
	metrics := noMetrics{}
	logger := discard()
	cfg := &config.Config{}

	// Common interaction skeleton
	base := func(options []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			Type:      discordgo.InteractionApplicationCommand,
			GuildID:   "g1",
			ChannelID: "c1",
			Member:    &discordgo.Member{User: &discordgo.User{ID: "u1"}},
			Data:      discordgo.ApplicationCommandInteractionData{Name: "claimtag", Options: options},
		}}
	}

	t.Run("success", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
			return nil
		}

		helper := &fakeHelpers{
			CreateNewMessageFunc: func(payload interface{}, topic string) (*message.Message, error) {
				return &message.Message{}, nil
			},
		}

		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}

		eb := &fakeEventBus{}
		store := &memStore{}

		mgr := NewClaimTagManager(sess, eb, logger, helper, cfg, resolver, store, nil, tracer, metrics)
		// valid tag option 5
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err != nil || res.Error != nil {
			t.Fatalf("expected success, got res=%v err=%v", res, err)
		}
		if len(eb.topics) == 0 {
			t.Fatalf("expected publish to be called")
		}
	})

	t.Run("resolver error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		helper := &fakeHelpers{}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return nil, errors.New("boom")
			},
		}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		res, err := mgr.HandleClaimTagCommand(ctx, base(nil))
		if err == nil || res.Error == nil {
			t.Fatalf("expected resolver error path")
		}
	})

	t.Run("no option provided", func(t *testing.T) {
		sess := discord.NewFakeSession()
		helper := &fakeHelpers{}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		res, err := mgr.HandleClaimTagCommand(ctx, base(nil))
		if err != nil || res.Error == nil {
			t.Fatalf("expected validation error for missing option")
		}
	})

	t.Run("invalid range option", func(t *testing.T) {
		sess := discord.NewFakeSession()
		helper := &fakeHelpers{}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(0)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err != nil || res.Error == nil {
			t.Fatalf("expected validation error for out-of-range tag")
		}
	})

	t.Run("interaction respond error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
			return errors.New("resp err")
		}
		helper := &fakeHelpers{}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from InteractionRespond")
		}
	})

	t.Run("store set error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		// Respond succeeds so we reach Set
		sess.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
			return nil
		}
		helper := &fakeHelpers{}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		store := &errStore{}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, store, nil, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from interaction store Set")
		}
	})

	t.Run("create message error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
			return nil
		}
		helper := &fakeHelpers{
			CreateNewMessageFunc: func(payload interface{}, topic string) (*message.Message, error) {
				return nil, errors.New("create err")
			},
		}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from CreateNewMessage")
		}
	})

	t.Run("publish error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
			return nil
		}
		helper := &fakeHelpers{
			CreateNewMessageFunc: func(payload interface{}, topic string) (*message.Message, error) {
				return &message.Message{}, nil
			},
		}
		resolver := &fakeGuildConfigResolver{
			GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
				return &storage.GuildConfig{}, nil
			},
		}
		eb := &fakeEventBus{err: errors.New("pub err")}
		mgr := NewClaimTagManager(sess, eb, logger, helper, cfg, resolver, &memStore{}, nil, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from Publish")
		}
	})
}

func TestUpdateInteractionResponse_Variants(t *testing.T) {
	ctx := context.Background()
	tracer := otel.Tracer("test")
	metrics := noMetrics{}
	logger := discard()
	cfg := &config.Config{}

	t.Run("not found", func(t *testing.T) {
		mgr := NewClaimTagManager(nil, &fakeEventBus{}, logger, nil, cfg, nil, &memStore{}, nil, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "missing", "msg")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error for missing correlation id")
		}
	})

	t.Run("bad stored type", func(t *testing.T) {
		store := &memStore{m: map[string]interface{}{"cid": 42}}
		mgr := NewClaimTagManager(nil, &fakeEventBus{}, logger, nil, cfg, nil, store, nil, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "msg")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error for wrong stored type")
		}
	})

	t.Run("edit error", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return nil, errors.New("edit err")
		}
		store := &memStore{m: map[string]interface{}{"cid": &discordgo.Interaction{}}}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, nil, cfg, nil, store, nil, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "done")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error from InteractionResponseEdit")
		}
	})

	t.Run("success", func(t *testing.T) {
		sess := discord.NewFakeSession()
		sess.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
			return &discordgo.Message{}, nil
		}
		store := &memStore{m: map[string]interface{}{"cid": &discordgo.Interaction{}}}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, nil, cfg, nil, store, nil, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "updated")
		if err != nil || res.Error != nil || res.Success == nil {
			t.Fatalf("expected success, got res=%v err=%v", res, err)
		}
		if _, err := store.Get(ctx, "cid"); err == nil {
			t.Fatalf("expected store to delete correlation id after success")
		}
	})
}

// errStore forces Set error
type errStore struct{ memStore }

func (e *errStore) Set(_ context.Context, id string, v interface{}) error {
	return errors.New("set err")
}
