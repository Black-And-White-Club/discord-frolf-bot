package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleGuildConfigUpdated(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful guild config updated - no role fields",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"updated_fields": ["signup_channel_id", "event_channel_id"]
				}`))
			}(),
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				// No Discord calls expected for non-role updates
			},
		},
		{
			name: "guild config updated with admin role field",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"updated_fields": ["admin_role_id", "signup_channel_id"]
				}`))
			}(),
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				// No Discord calls expected - role updates are just logged for now
			},
		},
		{
			name: "guild config updated with editor role field",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"updated_fields": ["editor_role_id"]
				}`))
			}(),
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				// No Discord calls expected - role updates are just logged for now
			},
		},
		{
			name: "guild config updated with user role field",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"updated_fields": ["user_role_id"]
				}`))
			}(),
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				// No Discord calls expected - role updates are just logged for now
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
			mockHelpers := util_mocks.NewMockHelpers(ctrl)

			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, nil))
			cfg := &config.Config{}
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			if tt.setup != nil {
				tt.setup(ctrl, mockGuildDiscord, mockHelpers)
			}

			h := &GuildHandlers{
				Logger:       logger,
				Config:       cfg,
				Helpers:      mockHelpers,
				GuildDiscord: mockGuildDiscord,
				Tracer:       tracer,
				Metrics:      metrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						payload := &guildevents.GuildConfigUpdatedPayloadV1{
							GuildID:       sharedtypes.GuildID("123456789"),
							UpdatedFields: []string{"signup_channel_id", "event_channel_id"},
						}

						// Override payload based on test case
						if bytes.Contains(msg.Payload, []byte("admin_role_id")) {
							payload.UpdatedFields = []string{"admin_role_id", "signup_channel_id"}
						} else if bytes.Contains(msg.Payload, []byte("editor_role_id")) {
							payload.UpdatedFields = []string{"editor_role_id"}
						} else if bytes.Contains(msg.Payload, []byte("user_role_id")) {
							payload.UpdatedFields = []string{"user_role_id"}
						}

						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigUpdated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigUpdated() = %v, want %v", got, tt.want)
			}

			// Verify appropriate log messages
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config updated")) {
				t.Errorf("Expected info log message not found in output: %s", logContent)
			}

			// Check for role update log if role fields were updated
			if bytes.Contains(tt.msg.Payload, []byte("role_id")) {
				if !bytes.Contains([]byte(logContent), []byte("Role configuration updated")) {
					t.Errorf("Expected role update log message not found in output: %s", logContent)
				}
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigUpdateFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
	}{
		{
			name: "guild config update failed - logs error and continues",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"reason": "database connection failed"
				}`))
			}(),
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
			mockHelpers := util_mocks.NewMockHelpers(ctrl)

			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, nil))
			cfg := &config.Config{}
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:       logger,
				Config:       cfg,
				Helpers:      mockHelpers,
				GuildDiscord: mockGuildDiscord,
				Tracer:       tracer,
				Metrics:      metrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						payload := &guildevents.GuildConfigUpdateFailedPayloadV1{
							GuildID: sharedtypes.GuildID("123456789"),
							Reason:  "database connection failed",
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigUpdateFailed(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigUpdateFailed() = %v, want %v", got, tt.want)
			}

			// Verify that error was logged
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config update failed")) {
				t.Errorf("Expected error log message not found in output: %s", logContent)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrieved(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
	}{
		{
			name: "guild config retrieved successfully - logs info and continues",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789"
				}`))
			}(),
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
			mockHelpers := util_mocks.NewMockHelpers(ctrl)

			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, nil))
			cfg := &config.Config{}
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:       logger,
				Config:       cfg,
				Helpers:      mockHelpers,
				GuildDiscord: mockGuildDiscord,
				Tracer:       tracer,
				Metrics:      metrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						payload := &guildevents.GuildConfigRetrievedPayloadV1{
							GuildID: sharedtypes.GuildID("123456789"),
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigRetrieved(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrieved() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigRetrieved() = %v, want %v", got, tt.want)
			}

			// Verify that info was logged
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config retrieved successfully")) {
				t.Errorf("Expected info log message not found in output: %s", logContent)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrievalFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
	}{
		{
			name: "guild config retrieval failed - logs error and continues",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"reason": "config not found"
				}`))
			}(),
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
			mockHelpers := util_mocks.NewMockHelpers(ctrl)

			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, nil))
			cfg := &config.Config{}
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:       logger,
				Config:       cfg,
				Helpers:      mockHelpers,
				GuildDiscord: mockGuildDiscord,
				Tracer:       tracer,
				Metrics:      metrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						payload := &guildevents.GuildConfigRetrievalFailedPayloadV1{
							GuildID: sharedtypes.GuildID("123456789"),
							Reason:  "config not found",
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigRetrievalFailed(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrievalFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigRetrievalFailed() = %v, want %v", got, tt.want)
			}

			// Verify that error was logged
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config retrieval failed")) {
				t.Errorf("Expected error log message not found in output: %s", logContent)
			}
		})
	}
}
