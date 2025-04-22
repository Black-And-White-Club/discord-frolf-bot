package deleteround

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewDeleteRoundManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager := NewDeleteRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)
	impl, ok := manager.(*deleteRoundManager)
	if !ok {
		t.Fatalf("Expected *deleteRoundManager, got %T", manager)
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
	if impl.interactionStore != mockInteractionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.tracer != tracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != mockMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
}

func Test_wrapDeleteRoundOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name       string
		operation  string
		fn         func(context.Context) (DeleteRoundOperationResult, error)
		expectErr  string
		expectRes  DeleteRoundOperationResult
		mockMetric func()
	}{
		{
			name:      "success path",
			operation: "delete_success",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: DeleteRoundOperationResult{Success: "yay!"},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "delete_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "delete_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "delete_error",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "delete_error operation error: bad op",
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "delete_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "delete_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "delete_result_error",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: DeleteRoundOperationResult{Error: errors.New("oopsie")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "delete_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "delete_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "delete_panic",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "",
			expectRes: DeleteRoundOperationResult{Error: nil},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "delete_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "delete_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil fn",
			operation: "delete_nil",
			fn:        nil,
			expectRes: DeleteRoundOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetric != nil {
				tt.mockMetric()
			}
			got, err := wrapDeleteRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, mockMetrics)

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Expected error to contain %q, got %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectRes.Success != nil && got.Success != tt.expectRes.Success {
				t.Errorf("Success = %v, want %v", got.Success, tt.expectRes.Success)
			}
			if tt.expectRes.Error != nil && (got.Error == nil || !strings.Contains(got.Error.Error(), tt.expectRes.Error.Error())) {
				t.Errorf("Error = %v, want %v", got.Error, tt.expectRes.Error)
			}
		})
	}
}
