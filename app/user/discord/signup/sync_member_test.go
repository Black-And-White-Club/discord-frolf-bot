package signup

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

// trackingEventBus wraps FakeEventBus to track published messages
type trackingEventBus struct {
	*testutils.FakeEventBus
	mu       sync.Mutex
	messages []*message.Message
}

func newTrackingEventBus() *trackingEventBus {
	t := &trackingEventBus{
		FakeEventBus: &testutils.FakeEventBus{},
		messages:     []*message.Message{},
	}
	t.PublishFunc = func(topic string, messages ...*message.Message) error {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.messages = append(t.messages, messages...)
		return nil
	}
	return t
}

func (t *trackingEventBus) PublishedMessages() []*message.Message {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*message.Message, len(t.messages))
	copy(out, t.messages)
	return out
}

func newTestHelper() *testutils.FakeHelpers {
	return &testutils.FakeHelpers{
		CreateNewMessageFunc: func(payload any, topic string) (*message.Message, error) {
			data, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			msg := message.NewMessage(watermill.NewUUID(), data)
			msg.Metadata.Set("topic", topic)
			return msg, nil
		},
	}
}

func TestSignupManager_SyncMember(t *testing.T) {
	tests := []struct {
		name         string
		guildID      string
		userID       string
		setupSession func(*discord.FakeSession)
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name:    "success with nickname",
			guildID: "guild-123",
			userID:  "user-456",
			setupSession: func(s *discord.FakeSession) {
				s.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{
						Nick: "ServerNickname",
						User: &discordgo.User{
							ID:       "user-456",
							Username: "testuser",
							Avatar:   "avatar-hash-123",
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "success without nickname",
			guildID: "guild-123",
			userID:  "user-456",
			setupSession: func(s *discord.FakeSession) {
				s.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{
						Nick: "", // No server nickname
						User: &discordgo.User{
							ID:       "user-456",
							Username: "testuser",
							Avatar:   "",
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "discord API error",
			guildID: "guild-123",
			userID:  "user-456",
			setupSession: func(s *discord.FakeSession) {
				s.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return nil, errors.New("discord API error: rate limited")
				}
			},
			wantErr:    true,
			wantErrMsg: "failed to fetch guild member",
		},
		{
			name:    "member is nil",
			guildID: "guild-123",
			userID:  "user-456",
			setupSession: func(s *discord.FakeSession) {
				s.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return nil, nil
				}
			},
			wantErr:    true,
			wantErrMsg: "member not found in guild",
		},
		{
			name:    "member.User is nil",
			guildID: "guild-123",
			userID:  "user-456",
			setupSession: func(s *discord.FakeSession) {
				s.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{
						Nick: "SomeNick",
						User: nil,
					}, nil
				}
			},
			wantErr:    true,
			wantErrMsg: "member not found in guild",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := &discord.FakeSession{}
			fakeEventBus := newTrackingEventBus()
			fakeInteractionStore := testutils.NewFakeStorage[any]()

			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}

			sm := &signupManager{
				session:             fakeSession,
				publisher:           fakeEventBus,
				logger:              slog.Default(),
				helper:              newTestHelper(),
				interactionStore:    fakeInteractionStore,
				guildConfigResolver: &guildconfig.FakeGuildConfigResolver{},
				tracer:              noop.NewTracerProvider().Tracer("test"),
				metrics:             &discordmetrics.NoOpMetrics{},
			}

			err := sm.SyncMember(context.Background(), tt.guildID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SyncMember() expected error but got nil")
					return
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SyncMember() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SyncMember() unexpected error: %v", err)
				}
				// Verify event was published on success
				if len(fakeEventBus.PublishedMessages()) == 0 {
					t.Error("expected event to be published but none were")
				}
			}
		})
	}
}

func TestGuildNickname(t *testing.T) {
	tests := []struct {
		name   string
		member *discordgo.Member
		want   string
	}{
		{
			name: "has nickname",
			member: &discordgo.Member{
				Nick: "ServerNickname",
				User: &discordgo.User{
					ID:       "user-123",
					Username: "originalname",
				},
			},
			want: "ServerNickname",
		},
		{
			name: "no nickname returns empty",
			member: &discordgo.Member{
				Nick: "",
				User: &discordgo.User{
					ID:       "user-123",
					Username: "originalname",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guildNickname(tt.member)
			if got != tt.want {
				t.Errorf("guildNickname() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublishUserProfile(t *testing.T) {
	tests := []struct {
		name        string
		member      *discordgo.Member
		guildID     string
		wantPublish bool
	}{
		{
			name: "publishes event successfully",
			member: &discordgo.Member{
				Nick: "TestNick",
				User: &discordgo.User{
					ID:       "user-123",
					Username: "testuser",
					Avatar:   "avatar-hash",
				},
			},
			guildID:     "guild-456",
			wantPublish: true,
		},
		{
			name:        "nil member does not publish",
			member:      nil,
			guildID:     "guild-456",
			wantPublish: false,
		},
		{
			name: "nil user does not publish",
			member: &discordgo.Member{
				Nick: "TestNick",
				User: nil,
			},
			guildID:     "guild-456",
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventBus := newTrackingEventBus()
			sm := &signupManager{
				session:          &discord.FakeSession{},
				publisher:        eventBus,
				logger:           slog.Default(),
				helper:           newTestHelper(),
				interactionStore: testutils.NewFakeStorage[any](),
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
			}

			sm.publishUserProfile(context.Background(), tt.member, tt.guildID)

			published := len(eventBus.PublishedMessages()) > 0
			if tt.wantPublish && !published {
				t.Error("expected event to be published but none were")
			}
			if !tt.wantPublish && published {
				t.Error("expected no event to be published but one was")
			}
		})
	}
}

func TestPublishRoleSync(t *testing.T) {
	const (
		adminRoleID      = "role-admin-111"
		editorRoleID     = "role-editor-222"
		registeredRoleID = "role-registered-333"
		guildID          = "guild-123"
		userID           = "user-456"
	)

	baseGuildCfg := &storage.GuildConfig{
		GuildID:          guildID,
		AdminRoleID:      adminRoleID,
		EditorRoleID:     editorRoleID,
		RegisteredRoleID: registeredRoleID,
	}

	newSM := func(resolver guildconfig.GuildConfigResolver, bus *trackingEventBus) *signupManager {
		return &signupManager{
			session:             &discord.FakeSession{},
			publisher:           bus,
			logger:              slog.Default(),
			helper:              newTestHelper(),
			interactionStore:    testutils.NewFakeStorage[any](),
			guildConfigResolver: resolver,
			tracer:              noop.NewTracerProvider().Tracer("test"),
			metrics:             &discordmetrics.NoOpMetrics{},
		}
	}

	tests := []struct {
		name        string
		memberRoles []string
		resolver    *guildconfig.FakeGuildConfigResolver
		wantPublish bool
		wantRole    sharedtypes.UserRoleEnum
	}{
		{
			name:        "admin role publishes UserRoleAdmin",
			memberRoles: []string{adminRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: true,
			wantRole:    sharedtypes.UserRoleAdmin,
		},
		{
			name:        "editor role publishes UserRoleEditor",
			memberRoles: []string{editorRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: true,
			wantRole:    sharedtypes.UserRoleEditor,
		},
		{
			name:        "registered role publishes UserRoleUser",
			memberRoles: []string{registeredRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: true,
			wantRole:    sharedtypes.UserRoleUser,
		},
		{
			name:        "admin wins over editor when both present",
			memberRoles: []string{editorRoleID, adminRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: true,
			wantRole:    sharedtypes.UserRoleAdmin,
		},
		{
			name:        "no recognized roles skips publish",
			memberRoles: []string{"role-unknown-999"},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: false,
		},
		{
			name:        "empty member roles skips publish",
			memberRoles: []string{},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return baseGuildCfg, nil
				},
			},
			wantPublish: false,
		},
		{
			name:        "guild config error skips publish without panic",
			memberRoles: []string{adminRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return nil, errors.New("backend unavailable")
				},
			},
			wantPublish: false,
		},
		{
			name:        "nil guild config skips publish without panic",
			memberRoles: []string{adminRoleID},
			resolver: &guildconfig.FakeGuildConfigResolver{
				GetGuildConfigWithContextFunc: func(_ context.Context, _ string) (*storage.GuildConfig, error) {
					return nil, nil
				},
			},
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := newTrackingEventBus()
			sm := newSM(tt.resolver, bus)

			sm.publishRoleSync(context.Background(), guildID, userID, tt.memberRoles)

			msgs := bus.PublishedMessages()
			if tt.wantPublish {
				if len(msgs) == 0 {
					t.Fatal("expected role sync event to be published but none were")
				}
				// Verify topic
				topic := msgs[0].Metadata.Get("topic")
				if topic != userevents.UserRoleUpdateRequestedV1 {
					t.Errorf("published topic = %q, want %q", topic, userevents.UserRoleUpdateRequestedV1)
				}
				// Verify payload role
				var payload userevents.UserRoleUpdateRequestedPayloadV1
				if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
					t.Fatalf("failed to unmarshal payload: %v", err)
				}
				if payload.Role != tt.wantRole {
					t.Errorf("payload.Role = %q, want %q", payload.Role, tt.wantRole)
				}
				if payload.UserID != sharedtypes.DiscordID(userID) {
					t.Errorf("payload.UserID = %q, want %q", payload.UserID, userID)
				}
				if payload.GuildID != sharedtypes.GuildID(guildID) {
					t.Errorf("payload.GuildID = %q, want %q", payload.GuildID, guildID)
				}
				if payload.RequesterID != "" {
					t.Errorf("payload.RequesterID = %q, want empty for system syncs", payload.RequesterID)
				}
			} else {
				if len(msgs) != 0 {
					t.Errorf("expected no role sync event but got %d", len(msgs))
				}
			}
		})
	}
}

func TestDeriveRole(t *testing.T) {
	const (
		adminRoleID      = "role-admin-111"
		editorRoleID     = "role-editor-222"
		registeredRoleID = "role-registered-333"
	)

	cfg := &storage.GuildConfig{
		AdminRoleID:      adminRoleID,
		EditorRoleID:     editorRoleID,
		RegisteredRoleID: registeredRoleID,
	}

	tests := []struct {
		name        string
		memberRoles []string
		cfg         *storage.GuildConfig
		want        sharedtypes.UserRoleEnum
	}{
		{
			name:        "admin role",
			memberRoles: []string{adminRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleAdmin,
		},
		{
			name:        "editor role",
			memberRoles: []string{editorRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleEditor,
		},
		{
			name:        "registered role",
			memberRoles: []string{registeredRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleUser,
		},
		{
			name:        "admin beats editor",
			memberRoles: []string{editorRoleID, adminRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleAdmin,
		},
		{
			name:        "admin beats registered",
			memberRoles: []string{registeredRoleID, adminRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleAdmin,
		},
		{
			name:        "editor beats registered",
			memberRoles: []string{registeredRoleID, editorRoleID},
			cfg:         cfg,
			want:        sharedtypes.UserRoleEditor,
		},
		{
			name:        "no matching role returns empty",
			memberRoles: []string{"role-unknown-999"},
			cfg:         cfg,
			want:        "",
		},
		{
			name:        "empty member roles returns empty",
			memberRoles: []string{},
			cfg:         cfg,
			want:        "",
		},
		{
			name:        "empty config role IDs never match",
			memberRoles: []string{adminRoleID},
			cfg:         &storage.GuildConfig{},
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveRole(tt.memberRoles, tt.cfg)
			if got != tt.want {
				t.Errorf("deriveRole() = %q, want %q", got, tt.want)
			}
		})
	}
}
