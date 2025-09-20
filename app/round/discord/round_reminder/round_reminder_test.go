package roundreminder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewRoundReminderManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := mocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	manager := NewRoundReminderManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockTracer, mockMetrics, mockGuildConfigResolver)
	impl, ok := manager.(*roundReminderManager)
	if !ok {
		t.Fatalf("Expected *roundReminderManager, got %T", manager)
	}

	if impl.session != mockSession {
		t.Error("Expected session to be assigned")
	}
	if impl.publisher != mockEventBus {
		t.Error("Expected publisher to be assigned")
	}
	if impl.logger != logger {
		t.Error("Expected logger to be assigned")
	}
	if impl.helper != mockHelper {
		t.Error("Expected helper to be assigned")
	}
	if impl.config != mockConfig {
		t.Error("Expected config to be assigned")
	}
	if impl.tracer != mockTracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != mockMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
}

func TestNewRoundReminderManager_WithNilLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockHelper := mocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	// Test with nil logger
	manager := NewRoundReminderManager(mockSession, mockEventBus, nil, mockHelper, mockConfig, mockTracer, mockMetrics, mockGuildConfigResolver)
	if manager == nil {
		t.Fatal("Expected manager to be created even with nil logger")
	}
}

func Test_wrapRoundReminderOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name       string
		operation  string
		fn         func(context.Context) (RoundReminderOperationResult, error)
		expectErr  string
		expectRes  RoundReminderOperationResult
		mockMetric func()
	}{
		{
			name:      "success path",
			operation: "reminder_success",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Success: "success"}, nil
			},
			expectRes: RoundReminderOperationResult{Success: "success"},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "reminder_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "reminder_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "reminder_error",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{}, errors.New("operation failed")
			},
			expectErr: "reminder_error operation error: operation failed",
			expectRes: RoundReminderOperationResult{Error: fmt.Errorf("reminder_error operation error: operation failed")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "reminder_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "reminder_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "reminder_result_error",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: RoundReminderOperationResult{Error: errors.New("result error")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "reminder_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "reminder_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "reminder_panic",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				panic("unexpected panic")
			},
			expectErr: "",                                       // The outer function returns nil error on panic recovery
			expectRes: RoundReminderOperationResult{Error: nil}, // The struct's Error field is nil on panic recovery before assignment
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "reminder_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "reminder_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil fn",
			operation: "reminder_nil",
			fn:        nil,
			expectRes: RoundReminderOperationResult{Error: errors.New("operation function is nil")},
		},
		{
			name:      "nil tracer",
			operation: "reminder_nil_tracer",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Success: "success with nil tracer"}, nil
			},
			expectRes: RoundReminderOperationResult{Success: "success with nil tracer"},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "reminder_nil_tracer", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "reminder_nil_tracer").Times(1)
			},
		},
		{
			name:      "nil metrics",
			operation: "reminder_nil_metrics",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Success: "success with nil metrics"}, nil
			},
			expectRes: RoundReminderOperationResult{Success: "success with nil metrics"},
			// No mockMetric function for nil metrics case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metricsToUse discordmetrics.DiscordMetrics
			if tt.name == "nil metrics" {
				metricsToUse = nil
			} else {
				metricsToUse = mockMetrics
				if tt.mockMetric != nil {
					tt.mockMetric()
				}
			}

			tracerToUse := tracer
			if tt.name == "nil tracer" {
				tracerToUse = nil
			}

			got, err := wrapRoundReminderOperation(context.Background(), tt.operation, tt.fn, logger, tracerToUse, metricsToUse)

			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("Expected error %q, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Compare the result struct fields individually, checking error string content
			if tt.expectRes.Success != nil && !reflect.DeepEqual(got.Success, tt.expectRes.Success) {
				t.Errorf("Success = %v, want %v", got.Success, tt.expectRes.Success)
			}
			if tt.expectRes.Failure != nil && !reflect.DeepEqual(got.Failure, tt.expectRes.Failure) {
				t.Errorf("Failure = %v, want %v", got.Failure, tt.expectRes.Failure)
			}

			if tt.expectRes.Error != nil {
				if got.Error == nil || !strings.Contains(got.Error.Error(), tt.expectRes.Error.Error()) {
					t.Errorf("Error = %v, want error containing %q", got.Error, tt.expectRes.Error.Error())
				}
			} else if got.Error != nil && tt.name != "panic recovery" {
				t.Errorf("Expected no error in result struct, got %v", got.Error)
			}
		})
	}
}
