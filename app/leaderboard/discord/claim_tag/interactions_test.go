package claimtag

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	gc_mocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.uber.org/mock/gomock"
)

// simple in-memory ISInterface stub
type memStore struct{ m map[string]interface{} }

func (s *memStore) Set(id string, v interface{}, _ time.Duration) error {
	if s.m == nil {
		s.m = map[string]interface{}{}
	}
	s.m[id] = v
	return nil
}

func (s *memStore) Delete(id string) {
	if s.m != nil {
		delete(s.m, id)
	}
}

func (s *memStore) Get(id string) (interface{}, bool) {
	if s.m == nil {
		return nil, false
	}
	v, ok := s.m[id]
	return v, ok
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

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)

		helper := util_mocks.NewMockHelpers(ctrl)
		helper.EXPECT().CreateNewMessage(gomock.Any(), gomock.Any()).Return(&message.Message{}, nil)

		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)

		eb := &fakeEventBus{}
		store := &memStore{}

		mgr := NewClaimTagManager(sess, eb, logger, helper, cfg, resolver, store, tracer, metrics)
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		helper := util_mocks.NewMockHelpers(ctrl)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(nil, errors.New("boom"))
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
		res, err := mgr.HandleClaimTagCommand(ctx, base(nil))
		if err == nil || res.Error == nil {
			t.Fatalf("expected resolver error path")
		}
	})

	t.Run("no option provided", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		helper := util_mocks.NewMockHelpers(ctrl)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
		res, err := mgr.HandleClaimTagCommand(ctx, base(nil))
		if err != nil || res.Error == nil {
			t.Fatalf("expected validation error for missing option")
		}
	})

	t.Run("invalid range option", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		helper := util_mocks.NewMockHelpers(ctrl)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(0)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err != nil || res.Error == nil {
			t.Fatalf("expected validation error for out-of-range tag")
		}
	})

	t.Run("interaction respond error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(errors.New("resp err"))
		helper := util_mocks.NewMockHelpers(ctrl)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from InteractionRespond")
		}
	})

	t.Run("store set error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		// Respond succeeds so we reach Set
		sess.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)
		helper := util_mocks.NewMockHelpers(ctrl)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		store := &errStore{}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, store, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from interaction store Set")
		}
	})

	t.Run("create message error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)
		helper := util_mocks.NewMockHelpers(ctrl)
		helper.EXPECT().CreateNewMessage(gomock.Any(), gomock.Any()).Return(nil, errors.New("create err"))
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
		opt := &discordgo.ApplicationCommandInteractionDataOption{Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
		res, err := mgr.HandleClaimTagCommand(ctx, base([]*discordgo.ApplicationCommandInteractionDataOption{opt}))
		if err == nil || res.Error == nil {
			t.Fatalf("expected error from CreateNewMessage")
		}
	})

	t.Run("publish error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)
		helper := util_mocks.NewMockHelpers(ctrl)
		helper.EXPECT().CreateNewMessage(gomock.Any(), gomock.Any()).Return(&message.Message{}, nil)
		resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
		resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{}, nil)
		eb := &fakeEventBus{err: errors.New("pub err")}
		mgr := NewClaimTagManager(sess, eb, logger, helper, cfg, resolver, &memStore{}, tracer, metrics)
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
		mgr := NewClaimTagManager(nil, &fakeEventBus{}, logger, nil, cfg, nil, &memStore{}, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "missing", "msg")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error for missing correlation id")
		}
	})

	t.Run("bad stored type", func(t *testing.T) {
		store := &memStore{m: map[string]interface{}{"cid": 42}}
		mgr := NewClaimTagManager(nil, &fakeEventBus{}, logger, nil, cfg, nil, store, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "msg")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error for wrong stored type")
		}
	})

	t.Run("edit error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Return(nil, errors.New("edit err"))
		store := &memStore{m: map[string]interface{}{"cid": &discordgo.Interaction{}}}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, nil, cfg, nil, store, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "done")
		if err != nil || res.Error == nil {
			t.Fatalf("expected error from InteractionResponseEdit")
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		sess := discordmocks.NewMockSession(ctrl)
		sess.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Return(&discordgo.Message{}, nil)
		store := &memStore{m: map[string]interface{}{"cid": &discordgo.Interaction{}}}
		mgr := NewClaimTagManager(sess, &fakeEventBus{}, logger, nil, cfg, nil, store, tracer, metrics)
		res, err := mgr.UpdateInteractionResponse(ctx, "cid", "updated")
		if err != nil || res.Error != nil || res.Success == nil {
			t.Fatalf("expected success, got res=%v err=%v", res, err)
		}
		if _, ok := store.Get("cid"); ok {
			t.Fatalf("expected store to delete correlation id after success")
		}
	})
}

// errStore forces Set error
type errStore struct{ memStore }

func (e *errStore) Set(id string, v interface{}, ttl time.Duration) error {
	return errors.New("set err")
}
