package handlers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleGuildSetupRequest(t *testing.T) {
	setupTime := time.Now()

	tests := []struct {
		name    string
		msg     *message.Message
		want    int // number of messages returned
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful guild setup request",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"guild_name": "Test Guild",
					"admin_user_id": "admin123",
					"signup_channel_id": "signup456",
					"event_channel_id": "event789",
					"leaderboard_channel_id": "leaderboard101",
					"registered_role_id": "user234",
					"admin_role_id": "admin567",
					"setup_completed_at": "2024-01-01T12:00:00Z"
				}`))
			}(),
			want:    1, // Should return one backend message
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockHelpers *util_mocks.MockHelpers) {
				backendMsg := message.NewMessage("2", []byte("backend payload"))
				backendMsg.Metadata.Set("guild_id", "123456789")

				mockHelpers.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						sharedevents.GuildConfigCreationRequested,
					).
					Return(backendMsg, nil).
					Times(1)
			},
		},
		{
			name: "failed to create backend message",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"guild_id": "123456789",
					"guild_name": "Test Guild",
					"admin_user_id": "admin123",
					"signup_channel_id": "signup456",
					"event_channel_id": "event789",
					"leaderboard_channel_id": "leaderboard101",
					"registered_role_id": "user234",
					"admin_role_id": "admin567",
					"setup_completed_at": "2024-01-01T12:00:00Z"
				}`))
			}(),
			want:    0,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockHelpers *util_mocks.MockHelpers) {
				mockHelpers.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						sharedevents.GuildConfigCreationRequested,
					).
					Return(nil, errors.New("failed to create message")).
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
						payload := &guildevents.GuildSetupEvent{
							GuildID:              "123456789",
							GuildName:            "Test Guild",
							AdminUserID:          "admin123",
							SignupChannelID:      "signup456",
							EventChannelID:       "event789",
							LeaderboardChannelID: "leaderboard101",
							RegisteredRoleID:     "user234",
							AdminRoleID:          "admin567",
							SetupCompletedAt:     setupTime,
						}
						return handlerFunc(context.Background(), msg, payload)
					}
				},
			}

			got, err := h.HandleGuildSetupRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildSetupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.want {
				t.Errorf("HandleGuildSetupRequest() returned %d messages, want %d", len(got), tt.want)
			}

			// Verify log messages
			logContent := logOutput.String()
			if !bytes.Contains([]byte(logContent), []byte("Processing guild setup request")) {
				t.Errorf("Expected initial log message not found in output: %s", logContent)
			}

			if tt.wantErr {
				if !bytes.Contains([]byte(logContent), []byte("Failed to create backend message")) {
					t.Errorf("Expected error log message not found in output: %s", logContent)
				}
			} else {
				if !bytes.Contains([]byte(logContent), []byte("Forwarding guild setup request to backend")) {
					t.Errorf("Expected forwarding log message not found in output: %s", logContent)
				}

				// Verify message has correct metadata
				if len(got) > 0 {
					guildID := got[0].Metadata.Get("guild_id")
					if guildID != "123456789" {
						t.Errorf("Expected guild_id metadata to be '123456789', got '%s'", guildID)
					}
				}
			}
		})
	}
}

func TestGuildSetupEventTransformation(t *testing.T) {
	// Test that the transformation from Discord event to backend payload is correct
	setupTime := time.Now()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
	mockHelpers := util_mocks.NewMockHelpers(ctrl)

	// Capture the payload passed to CreateResultMessage
	var capturedPayload interface{}
	mockHelpers.EXPECT().
		CreateResultMessage(
			gomock.Any(),
			gomock.Any(),
			sharedevents.GuildConfigCreationRequested,
		).
		DoAndReturn(func(msg *message.Message, payload interface{}, topic string) (*message.Message, error) {
			capturedPayload = payload
			backendMsg := message.NewMessage("2", []byte("test"))
			return backendMsg, nil
		}).
		Times(1)

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
				payload := &guildevents.GuildSetupEvent{
					GuildID:              "123456789",
					GuildName:            "Test Guild",
					AdminUserID:          "admin123",
					SignupChannelID:      "signup456",
					EventChannelID:       "event789",
					LeaderboardChannelID: "leaderboard101",
					RegisteredRoleID:     "user234",
					AdminRoleID:          "admin567",
					SetupCompletedAt:     setupTime,
				}
				return handlerFunc(context.Background(), msg, payload)
			}
		},
	}

	msg := message.NewMessage("1", []byte("test"))
	_, err := h.HandleGuildSetupRequest(msg)
	if err != nil {
		t.Fatalf("HandleGuildSetupRequest() failed: %v", err)
	}

	// Verify the payload transformation
	backendPayload, ok := capturedPayload.(sharedevents.GuildConfigRequestedPayload)
	if !ok {
		t.Fatalf("Expected payload to be GuildConfigRequestedPayload, got %T", capturedPayload)
	}

	expectedPayload := sharedevents.GuildConfigRequestedPayload{
		GuildID:              "123456789",
		SignupChannelID:      "signup456",
		EventChannelID:       "event789",
		LeaderboardChannelID: "leaderboard101",
		UserRoleID:           "user234",
		AdminRoleID:          "admin567",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
	}

	if !cmp.Equal(backendPayload, expectedPayload) {
		t.Errorf("Payload transformation mismatch:\n%s", cmp.Diff(expectedPayload, backendPayload))
	}
}
