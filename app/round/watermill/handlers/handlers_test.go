package roundhandlers

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
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

func TestNewRoundHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRoundDiscord := rounddiscord.NewMockRoundDiscordInterface(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewRoundHandlers(logger, cfg, mockHelpers, mockRoundDiscord, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil RoundHandlers")
		}

		rh := handlers.(*RoundHandlers)

		if rh.RoundDiscord != mockRoundDiscord {
			t.Errorf("RoundDiscord not set correctly")
		}
		if rh.Helpers != mockHelpers {
			t.Errorf("Helpers not set correctly")
		}
		if rh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if rh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if rh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if rh.Tracer != tracer {
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
					handlerName: "RoundNoPayload",
					unmarshalTo: nil, // This is correct for no payload
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
					Name string `json:"name"`
				}
				return args{
					handlerName: "RoundWithPayload",
					unmarshalTo: &TestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						if p, ok := payload.(*TestPayload); ok && p.Name == "Test" {
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
					reflect.ValueOf(out).Elem().FieldByName("Name").SetString("Test")
				})
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				return args{
					handlerName: "RoundHandlerError",
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
					handlerName: "RoundUnmarshalFail",
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

			wrapped := wrapHandler(
				testArgs.handlerName,
				testArgs.unmarshalTo,
				testArgs.handlerFunc,
				testArgs.logger,
				testArgs.metrics,
				testArgs.tracer,
				testArgs.helpers,
			)

			msg := message.NewMessage("id", []byte(`{"name": "Test"}`))
			_, err := wrapped(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("wrapHandler error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
