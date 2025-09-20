package handlers

import (
	"bytes"
	"context"
	"errors"
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

func TestGuildHandlers_HandleGuildConfigDeleted(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful guild config deleted",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789"
				}`))
			}(),
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				mockGuildDiscord.EXPECT().
					UnregisterAllCommands("123456789").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to unregister commands",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789"
				}`))
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				mockGuildDiscord.EXPECT().
					UnregisterAllCommands("123456789").
					Return(errors.New("failed to unregister commands")).
					Times(1)
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
						payload := &guildevents.GuildConfigDeletedPayload{
							GuildID: sharedtypes.GuildID("123456789"),
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigDeleted(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigDeleted() = %v, want %v", got, tt.want)
			}

			// Verify appropriate log messages
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config deleted")) {
				t.Errorf("Expected info log message not found in output: %s", logContent)
			}

			// Check for success or error log based on test case
			if tt.wantErr {
				if !bytes.Contains([]byte(logContent), []byte("Failed to unregister all commands")) {
					t.Errorf("Expected error log message not found in output: %s", logContent)
				}
			} else {
				if !bytes.Contains([]byte(logContent), []byte("Successfully unregistered all commands")) {
					t.Errorf("Expected success log message not found in output: %s", logContent)
				}
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigDeletionFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
	}{
		{
			name: "guild config deletion failed - logs warning and continues",
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
						payload := &guildevents.GuildConfigDeletionFailedPayload{
							GuildID: sharedtypes.GuildID("123456789"),
							Reason:  "database connection failed",
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigDeletionFailed(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigDeletionFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigDeletionFailed() = %v, want %v", got, tt.want)
			}

			// Verify that warning was logged
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config deletion failed")) {
				t.Errorf("Expected warning log message not found in output: %s", logContent)
			}
		})
	}
}
