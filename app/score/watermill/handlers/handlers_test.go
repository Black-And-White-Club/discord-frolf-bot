package scorehandlers

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	mockdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSession := mockdiscord.NewMockSession(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewScoreHandlers(logger, cfg, mockSession, mockHelpers, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil ScoreHandlers")
		}

		sh := handlers

		if sh.Session != mockSession {
			t.Errorf("Session not set correctly")
		}
		if sh.Helper != mockHelpers {
			t.Errorf("Helpers not set correctly")
		}
		if sh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if sh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if sh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if sh.Tracer != tracer {
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
					handlerName: "ScoreNoPayload",
					unmarshalTo: nil,
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
				a.metrics.(*mocks.MockDiscordMetrics).EXPECT().RecordHandlerSuccess(gomock.Any(), a.handlerName)
			},
		},
		{
			name: "Successful with payload",
			args: func(ctrl *gomock.Controller) args {
				type TestPayload struct {
					Score string `json:"score"`
				}
				return args{
					handlerName: "ScoreWithPayload",
					unmarshalTo: &TestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						if p, ok := payload.(*TestPayload); ok && p.Score == "42" {
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
					reflect.ValueOf(out).Elem().FieldByName("Score").SetString("42")
				})
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				return args{
					handlerName: "ScoreHandlerError",
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
				type BrokenPayload struct {
					X string `json:"x"`
				}
				return args{
					handlerName: "ScoreUnmarshalFail",
					unmarshalTo: &BrokenPayload{},
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

			var factory func() interface{}
			if testArgs.unmarshalTo != nil {
				if fn, ok := testArgs.unmarshalTo.(func() interface{}); ok {
					factory = fn
				} else {
					factory = func() interface{} { return utils.NewInstance(testArgs.unmarshalTo) }
				}
			}

			wrapped := wrapHandler(
				testArgs.handlerName,
				factory,
				testArgs.handlerFunc,
				testArgs.logger,
				testArgs.metrics,
				testArgs.tracer,
				testArgs.helpers,
			)

			msg := message.NewMessage("id", []byte(`{"score": "42"}`))
			_, err := wrapped(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("wrapHandler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
