package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
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

func TestGuildHandlers_HandleGuildConfigCreated(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful guild config created",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"config_id": "config_123",
					"created_at": "2024-01-01T12:00:00Z"
				}`))
			}(),
			want:    nil, // Handler returns nil on success
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				mockGuildDiscord.EXPECT().
					RegisterAllCommands("123456789").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to register commands",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"config_id": "config_123", 
					"created_at": "2024-01-01T12:00:00Z"
				}`))
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, _ *util_mocks.MockHelpers) {
				mockGuildDiscord.EXPECT().
					RegisterAllCommands("123456789").
					Return(errors.New("failed to register commands")).
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

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
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
						payload := &guildevents.GuildConfigCreatedPayload{
							GuildID: sharedtypes.GuildID("123456789"),
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigCreated(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigCreated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
	}{
		{
			name: "guild config creation failed - logs warning and continues",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"reason": "database connection failed"
				}`))
			}(),
			want:    nil, // Handler returns nil (no error, just logs)
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
						payload := &guildevents.GuildConfigCreationFailedPayload{
							GuildID: sharedtypes.GuildID("123456789"),
							Reason:  "database connection failed",
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildConfigCreationFailed(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("HandleGuildConfigCreationFailed() = %v, want %v", got, tt.want)
			}

			// Verify that warning was logged
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Guild config creation failed")) {
				t.Errorf("Expected warning log message not found in output: %s", logContent)
			}
		})
	}
}
