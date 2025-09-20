package leaderboardhandlers

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLeaderboardDiscord := leaderboarddiscord.NewMockLeaderboardDiscordInterface(ctrl)
		mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewLeaderboardHandlers(logger, cfg, mockHelpers, mockLeaderboardDiscord, mockGuildConfigResolver, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil LeaderboardHandlers")
		}

		lh := handlers.(*LeaderboardHandlers)

		if lh.LeaderboardDiscord != mockLeaderboardDiscord {
			t.Errorf("LeaderboardDiscord not set correctly")
		}
		if lh.GuildConfigResolver != mockGuildConfigResolver {
			t.Errorf("GuildConfigResolver not set correctly")
		}
		if lh.Helpers != mockHelpers {
			t.Errorf("Helpers not set correctly")
		}
		if lh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if lh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if lh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if lh.Tracer != tracer {
			t.Errorf("Tracer not set correctly")
		}
	})
}

func Test_wrapHandler(t *testing.T) {
	type args struct {
		handlerName string
		unmarshalTo interface{}
		handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)
		logger      *slog.Logger
		metrics     discordmetrics.DiscordMetrics
		tracer      trace.Tracer
		helpers     *mockHelpers.MockHelpers
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		wantErr bool
		setup   func(a *args, ctrl *gomock.Controller)
	}{
		{
			name: "Successful no-payload execution",
			args: func(ctrl *gomock.Controller) args {
				return args{
					handlerName: "LeaderboardNoPayload",
					unmarshalTo: nil, // No payload
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  loggerfrolfbot.NoOpLogger,
					metrics: mocks.NewMockDiscordMetrics(ctrl),
					tracer:  noop.NewTracerProvider().Tracer("test"),
					helpers: mockHelpers.NewMockHelpers(ctrl),
				}
			},
			wantErr: false,
			setup: func(a *args, ctrl *gomock.Controller) {
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerAttempt(gomock.Any(), a.handlerName)
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerDuration(gomock.Any(), a.handlerName, gomock.Any())
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerSuccess(gomock.Any(), a.handlerName) // Expect success
			},
		},
		{
			name: "Successful with payload",
			args: func(ctrl *gomock.Controller) args {
				type TestPayload struct {
					LeaderboardID string `json:"leaderboard_id"`
				}
				return args{
					handlerName: "LeaderboardWithPayload",
					unmarshalTo: &TestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						if p, ok := payload.(*TestPayload); ok && p.LeaderboardID == "123" {
							return []*message.Message{msg}, nil
						}
						return nil, errors.New("unexpected payload")
					},
					logger:  loggerfrolfbot.NoOpLogger,
					metrics: mocks.NewMockDiscordMetrics(ctrl),
					tracer:  noop.NewTracerProvider().Tracer("test"),
					helpers: mockHelpers.NewMockHelpers(ctrl),
				}
			},
			wantErr: false,
			setup: func(a *args, ctrl *gomock.Controller) {
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerAttempt(gomock.Any(), a.handlerName)
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerDuration(gomock.Any(), a.handlerName, gomock.Any())
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerSuccess(gomock.Any(), a.handlerName)

				a.helpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(nil).Do(func(_ *message.Message, out interface{}) {
					reflect.ValueOf(out).Elem().FieldByName("LeaderboardID").SetString("123")
				})
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				return args{
					handlerName: "LeaderboardHandlerError",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return nil, errors.New("handler failed")
					},
					logger:  loggerfrolfbot.NoOpLogger,
					metrics: mocks.NewMockDiscordMetrics(ctrl),
					tracer:  noop.NewTracerProvider().Tracer("test"),
					helpers: mockHelpers.NewMockHelpers(ctrl),
				}
			},
			wantErr: true,
			setup: func(a *args, ctrl *gomock.Controller) {
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerAttempt(gomock.Any(), a.handlerName)
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerDuration(gomock.Any(), a.handlerName, gomock.Any())
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerFailure(gomock.Any(), a.handlerName)
			},
		},
		{
			name: "Unmarshal error",
			args: func(ctrl *gomock.Controller) args {
				type LeaderboardPayload struct {
					UserID string `json:"user_id"`
				}
				return args{
					handlerName: "LeaderboardUnmarshalFail",
					unmarshalTo: &LeaderboardPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  loggerfrolfbot.NoOpLogger,
					metrics: mocks.NewMockDiscordMetrics(ctrl),
					tracer:  noop.NewTracerProvider().Tracer("test"),
					helpers: mockHelpers.NewMockHelpers(ctrl),
				}
			},
			wantErr: true,
			setup: func(a *args, ctrl *gomock.Controller) {
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerAttempt(gomock.Any(), a.handlerName)
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerDuration(gomock.Any(), a.handlerName, gomock.Any())
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerFailure(gomock.Any(), a.handlerName)

				a.helpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal fail"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testArgs := tt.args(ctrl)
			tt.setup(&testArgs, ctrl)

			wrapped := wrapHandler(
				testArgs.handlerName,
				testArgs.unmarshalTo,
				testArgs.handlerFunc,
				testArgs.logger,
				testArgs.metrics,
				testArgs.tracer,
				testArgs.helpers,
			)

			msg := message.NewMessage("id", []byte(`{"leaderboard_id": "123", "user_id": "456"}`))
			_, err := wrapped(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("wrapHandler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
