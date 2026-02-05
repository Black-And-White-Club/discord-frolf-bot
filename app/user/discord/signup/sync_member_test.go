package signup

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
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
				session:          fakeSession,
				publisher:        fakeEventBus,
				logger:           slog.Default(),
				interactionStore: fakeInteractionStore,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
			}

			err := sm.SyncMember(context.Background(), tt.guildID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SyncMember() expected error but got nil")
					return
				}
				if tt.wantErrMsg != "" && !containsString(err.Error(), tt.wantErrMsg) {
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

func TestResolveDisplayName(t *testing.T) {
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
			got := resolveDisplayName(tt.member)
			if got != tt.want {
				t.Errorf("resolveDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublishUserProfileFromMember(t *testing.T) {
	tests := []struct {
		name        string
		member      *discordgo.Member
		guildID     string
		useEventBus bool
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
			useEventBus: true,
			wantPublish: true,
		},
		{
			name:        "nil member does not publish",
			member:      nil,
			guildID:     "guild-456",
			useEventBus: true,
			wantPublish: false,
		},
		{
			name: "nil user does not publish",
			member: &discordgo.Member{
				Nick: "TestNick",
				User: nil,
			},
			guildID:     "guild-456",
			useEventBus: true,
			wantPublish: false,
		},
		{
			name: "nil eventBus does not crash",
			member: &discordgo.Member{
				Nick: "TestNick",
				User: &discordgo.User{
					ID:       "user-123",
					Username: "testuser",
				},
			},
			guildID:     "guild-456",
			useEventBus: false,
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.Default()

			if tt.useEventBus {
				eventBus := newTrackingEventBus()
				publishUserProfileFromMember(ctx, eventBus, logger, tt.member, tt.guildID)
				if tt.wantPublish {
					if len(eventBus.PublishedMessages()) == 0 {
						t.Error("expected event to be published but none were")
					}
				}
			} else {
				// Pass explicit nil to avoid typed nil pointer issue
				publishUserProfileFromMember(ctx, nil, logger, tt.member, tt.guildID)
				// No assertion needed - just verifying no panic
			}
		})
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
