package userhandlers

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	mockdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
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

func TestNewUserHandlers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates handlers with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Create mock dependencies
				mockUserDiscord := mockdiscord.NewMockUserDiscordInterface(ctrl)
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)
				mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
				cfg := &config.Config{}

				// Call the function being tested
				handlers := NewUserHandlers(logger, cfg, mockHelpers, mockUserDiscord, tracer, mockMetrics)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewUserHandlers returned nil")
				}

				// Access userHandlers directly from the UserHandler interface
				userHandlers := handlers.(*UserHandlers)

				// Override handlerWrapper to prevent unwanted tracing/logging/metrics calls during this test
				userHandlers.handlerWrapper = func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return func(msg *message.Message) ([]*message.Message, error) {
						// Directly call the handler function without any additional logic
						return handlerFunc(context.Background(), msg, unmarshalTo)
					}
				}

				// Check that all dependencies were correctly assigned
				if userHandlers.UserDiscord != mockUserDiscord {
					t.Errorf("UserDiscord not correctly assigned")
				}
				if userHandlers.Logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if userHandlers.Tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				if userHandlers.Helper != mockHelpers {
					t.Errorf("helpers not correctly assigned")
				}
				if userHandlers.Metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if userHandlers.Config != cfg {
					t.Errorf("config not correctly assigned")
				}

				// Ensure handlerWrapper is correctly set
				if userHandlers.handlerWrapper == nil {
					t.Errorf("handlerWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Call with nil dependencies
				handlers := NewUserHandlers(nil, nil, nil, nil, nil, nil)

				// Ensure handlers are correctly created
				if handlers == nil {
					t.Fatalf("NewUserHandlers returned nil")
				}

				// Check nil fields
				if uh, ok := handlers.(*UserHandlers); ok {
					if uh.UserDiscord != nil {
						t.Errorf("UserDiscord should be nil")
					}
					if uh.Logger != nil {
						t.Errorf("logger should be nil")
					}
					if uh.Tracer != nil {
						t.Errorf("tracer should be nil")
					}
					if uh.Helper != nil {
						t.Errorf("helpers should be nil")
					}
					if uh.Metrics != nil {
						t.Errorf("metrics should be nil")
					}
					if uh.Config != nil {
						t.Errorf("config should be nil")
					}

					// Ensure handlerWrapper is still set
					if uh.handlerWrapper == nil {
						t.Errorf("handlerWrapper should not be nil")
					}
				} else {
					t.Errorf("handlers is not of type *userHandlers")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestUserHandlers_wrapHandler(t *testing.T) {
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
		setup   func(a *args, ctrl *gomock.Controller) // Setup expectations per test
	}{
		{
			name: "Successful handler execution with no payload",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args, ctrl *gomock.Controller) {
				mockMetrics := a.metrics.(*mocks.MockDiscordMetrics)

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")
			},
		},
		{
			name: "Successful handler execution with payload",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				type TestPayload struct {
					Data string `json:"data"`
				}

				return args{
					handlerName: "testHandler",
					unmarshalTo: &TestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						if p, ok := payload.(*TestPayload); ok && p.Data == "test" {
							return []*message.Message{msg}, nil
						}
						return nil, errors.New("payload mismatch")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: false,
			setup: func(a *args, ctrl *gomock.Controller) {
				mockMetrics := a.metrics.(*mocks.MockDiscordMetrics)
				mockHelpers := a.helpers

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "testHandler")

				// Mock unmarshal payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(nil).Do(func(_ *message.Message, target interface{}) {
					reflect.ValueOf(target).Elem().FieldByName("Data").SetString("test")
				})
			},
		},
		{
			name: "Handler returns error",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				return args{
					handlerName: "testHandler",
					unmarshalTo: nil,
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return nil, errors.New("handler error")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args, ctrl *gomock.Controller) {
				mockMetrics := a.metrics.(*mocks.MockDiscordMetrics)

				// Use gomock.Any() for context parameter
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")
			},
		},
		{
			name: "Unmarshal payload fails",
			args: func(ctrl *gomock.Controller) args {
				logger := loggerfrolfbot.NoOpLogger
				tracer := noop.NewTracerProvider().Tracer("test")
				mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
				mockHelpers := mockHelpers.NewMockHelpers(ctrl)

				type TestPayload struct {
					Data string `json:"data"`
				}

				return args{
					handlerName: "testHandler",
					unmarshalTo: &TestPayload{},
					handlerFunc: func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
						return []*message.Message{msg}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
					helpers: mockHelpers,
				}
			},
			wantErr: true,
			setup: func(a *args, ctrl *gomock.Controller) {
				mockMetrics := a.metrics.(*mocks.MockDiscordMetrics)
				mockHelpers := a.helpers

				// Use gomock.Any() for context parameter
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "testHandler")
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "testHandler", gomock.Any())
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "testHandler")

				// Use gomock.Any() for message and payload parameters
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Initialize args using fresh mock controller
			testArgs := tt.args(ctrl)

			// Set up mock expectations
			tt.setup(&testArgs, ctrl)

			// Create a userHandlers instance using the constructor
			uh := NewUserHandlers(
				testArgs.logger,
				&config.Config{}, // Provide a non-nil config
				testArgs.helpers,
				&mockdiscord.MockUserDiscordInterface{}, // We don't directly test UserDiscord here
				testArgs.tracer,
				testArgs.metrics,
			).(*UserHandlers) // Assert to the concrete type

			// Now you can access the handlerWrapper
			handlerFunc := uh.handlerWrapper(
				testArgs.handlerName,
				testArgs.unmarshalTo,
				testArgs.handlerFunc,
			)

			msg := message.NewMessage("test-id", []byte(`{"data": "test"}`))
			_, err := handlerFunc(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.wrapHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
