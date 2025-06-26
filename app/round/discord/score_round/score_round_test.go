package scoreround

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreRoundManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)

	manager := NewScoreRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockTracer, mockMetrics)
	impl, ok := manager.(*scoreRoundManager)
	if !ok {
		t.Fatalf("Expected *scoreRoundManager, got %T", manager)
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

func Test_wrapScoreRoundOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name        string
		operation   string
		fn          func(context.Context) (ScoreRoundOperationResult, error)
		expectErr   string
		expectRes   ScoreRoundOperationResult
		mockMetrics func()
	}{
		{
			name:      "success path",
			operation: "handle_success",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Success: "success"}, nil
			},
			expectRes: ScoreRoundOperationResult{Success: "success"},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "handle_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "handle_error",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{}, errors.New("operation failed")
			},
			expectErr: "handle_error operation error: operation failed",
			expectRes: ScoreRoundOperationResult{Error: errors.New("handle_error operation error: operation failed")},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "handle_result_error",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				return ScoreRoundOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: ScoreRoundOperationResult{Error: errors.New("result error")},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "handle_panic",
			fn: func(ctx context.Context) (ScoreRoundOperationResult, error) {
				panic("unexpected panic")
			},
			expectRes: ScoreRoundOperationResult{Error: nil},
			mockMetrics: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "handle_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "handle_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil fn",
			operation: "handle_nil",
			fn:        nil,
			expectRes: ScoreRoundOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetrics != nil {
				tt.mockMetrics()
			}
			got, err := wrapScoreRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, mockMetrics)

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Expected error to contain %q, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Compare results, handling potential nil errors gracefully
			if tt.expectRes.Success != nil && !reflect.DeepEqual(got.Success, tt.expectRes.Success) {
				t.Errorf("Success = %v, want %v", got.Success, tt.expectRes.Success)
			}
			if tt.expectRes.Error != nil {
				if got.Error == nil || !strings.Contains(got.Error.Error(), tt.expectRes.Error.Error()) {
					t.Errorf("Error = %v, want %v", got.Error, tt.expectRes.Error)
				}
			} else if got.Error != nil {
				t.Errorf("Expected no error in result, got %v", got.Error)
			}
		})
	}
}
