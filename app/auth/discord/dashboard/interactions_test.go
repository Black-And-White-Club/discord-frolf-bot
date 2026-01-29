package dashboard

import (
	"context"
	"errors"
	"testing"

	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestDashboardManager_HandleDashboardCommand(t *testing.T) {
	tests := []struct {
		name          string
		interaction   *discordgo.InteractionCreate
		wantErr       bool
		setupSession  func(*discordpkg.FakeSession)
		setupEventBus func(*testutils.FakeEventBus)
		setupStore    func(*FakeInteractionStore)
		setupResolver func(*testutils.FakeGuildConfigResolver)
		setupHelper   func(*testutils.FakeHelpers)
		verifySession func(*testing.T, *discordpkg.FakeSession)
	}{
		{
			name: "successful dashboard command - publishes magic link request",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Token:   "test-token",
					GuildID: "guild-123",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "user-123"},
						Roles: []string{"player-role"},
					},
				},
			},
			wantErr: false,
			setupSession: func(session *discordpkg.FakeSession) {
				// Default behavior for interaction respond
			},
			setupResolver: func(resolver *testutils.FakeGuildConfigResolver) {
				resolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
					return &storage.GuildConfig{
						RegisteredRoleID: "player-role",
					}, nil
				}
			},
			setupHelper: func(helper *testutils.FakeHelpers) {
				helper.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
					return message.NewMessage("test-msg-id", []byte("{}")), nil
				}
			},
			setupEventBus: func(eb *testutils.FakeEventBus) {
				// Default behavior - Publish succeeds
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "InteractionRespond") {
					t.Error("expected InteractionRespond to acknowledge command")
				}
			},
		},
		{
			name: "guild config error - sends error followup",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Token:   "test-token",
					GuildID: "guild-123",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "user-123"},
						Roles: []string{},
					},
				},
			},
			wantErr: false, // Error is sent as followup, not returned
			setupResolver: func(resolver *testutils.FakeGuildConfigResolver) {
				resolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
					return nil, errors.New("guild not configured")
				}
			},
			setupHelper: func(helper *testutils.FakeHelpers) {
				helper.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
					return message.NewMessage("test-msg-id", []byte("{}")), nil
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "InteractionRespond") {
					t.Error("expected InteractionRespond to acknowledge command")
				}
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected FollowupMessageCreate with error message")
				}
			},
		},
		{
			name: "interaction respond error - returns error",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Token:   "test-token",
					GuildID: "guild-123",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "user-123"},
						Roles: []string{},
					},
				},
			},
			wantErr: true,
			setupSession: func(session *discordpkg.FakeSession) {
				session.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return errors.New("Discord API error")
				}
			},
			setupHelper: func(helper *testutils.FakeHelpers) {
				helper.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
					return message.NewMessage("test-msg-id", []byte("{}")), nil
				}
			},
		},
		{
			name: "interaction store error - sends error followup",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Token:   "test-token",
					GuildID: "guild-123",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "user-123"},
						Roles: []string{},
					},
				},
			},
			wantErr: false,
			setupResolver: func(resolver *testutils.FakeGuildConfigResolver) {
				resolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
					return &storage.GuildConfig{}, nil
				}
			},
			setupHelper: func(helper *testutils.FakeHelpers) {
				helper.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
					return message.NewMessage("test-msg-id", []byte("{}")), nil
				}
			},
			setupStore: func(store *FakeInteractionStore) {
				store.SetFunc = func(ctx context.Context, correlationID string, interaction any) error {
					return errors.New("store full")
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected error followup message")
				}
			},
		},
		{
			name: "publish error - sends error followup and cleans up stored interaction",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Token:   "test-token",
					GuildID: "guild-123",
					Member: &discordgo.Member{
						User:  &discordgo.User{ID: "user-123"},
						Roles: []string{},
					},
				},
			},
			wantErr: false,
			setupResolver: func(resolver *testutils.FakeGuildConfigResolver) {
				resolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
					return &storage.GuildConfig{}, nil
				}
			},
			setupHelper: func(helper *testutils.FakeHelpers) {
				helper.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
					return message.NewMessage("test-msg-id", []byte("{}")), nil
				}
			},
			setupEventBus: func(eb *testutils.FakeEventBus) {
				eb.PublishFunc = func(topic string, messages ...*message.Message) error {
					return errors.New("NATS connection failed")
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected error followup message on publish failure")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fakes
			fakeSession := discordpkg.NewFakeSession()
			fakeEventBus := &testutils.FakeEventBus{}
			fakeStore := NewFakeInteractionStore()
			fakeResolver := &testutils.FakeGuildConfigResolver{}
			fakeHelper := &testutils.FakeHelpers{}
			fakeMetrics := &testutils.FakeDiscordMetrics{}
			tracer := noop.NewTracerProvider().Tracer("test")

			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}
			if tt.setupEventBus != nil {
				tt.setupEventBus(fakeEventBus)
			}
			if tt.setupStore != nil {
				tt.setupStore(fakeStore)
			}
			if tt.setupResolver != nil {
				tt.setupResolver(fakeResolver)
			}
			if tt.setupHelper != nil {
				tt.setupHelper(fakeHelper)
			}

			// Create dashboard manager
			logger := loggerfrolfbot.NoOpLogger
			cfg := &config.Config{}
			permMapper := NewFakePermissionMapper()

			manager := NewDashboardManager(
				fakeSession,
				fakeEventBus,
				logger,
				cfg,
				tracer,
				fakeMetrics,
				fakeResolver,
				permMapper,
				fakeStore,
				fakeHelper,
			)

			// Execute
			err := manager.HandleDashboardCommand(context.Background(), tt.interaction)

			// Verify
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleDashboardCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.verifySession != nil {
				tt.verifySession(t, fakeSession)
			}
		})
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
