package startround

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
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewStartRoundManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := mocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockGuildConfigCache := storagemocks.NewMockISInterface[storage.GuildConfig](ctrl)

	mockTracer := noop.NewTracerProvider().Tracer("test")
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	mockGuildConfig := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	manager := NewStartRoundManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, mockGuildConfigCache, mockTracer, mockMetrics, mockGuildConfig)
	impl, ok := manager.(*startRoundManager)
	if !ok {
		t.Fatalf("Expected *startRoundManager, got %T", manager)
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

func Test_wrapStartRoundOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name       string
		operation  string
		fn         func(context.Context) (StartRoundOperationResult, error)
		expectErr  string
		expectRes  StartRoundOperationResult
		mockMetric func()
	}{
		{
			name:      "success path",
			operation: "start_success",
			fn: func(ctx context.Context) (StartRoundOperationResult, error) {
				return StartRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: StartRoundOperationResult{Success: "yay!"},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "start_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "start_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "start_error",
			fn: func(ctx context.Context) (StartRoundOperationResult, error) {
				return StartRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "start_error operation error: bad op",
			expectRes: StartRoundOperationResult{Error: fmt.Errorf("start_error operation error: bad op")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "start_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "start_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "start_result_error",
			fn: func(ctx context.Context) (StartRoundOperationResult, error) {
				return StartRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: StartRoundOperationResult{Error: errors.New("oopsie")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "start_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "start_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "start_panic",
			fn: func(ctx context.Context) (StartRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "",                                    // The outer function returns nil error on panic recovery
			expectRes: StartRoundOperationResult{Error: nil}, // The struct's Error field is nil on panic recovery before assignment
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "start_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "start_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil fn",
			operation: "start_nil",
			fn:        nil,
			expectErr: "operation function is nil",
			expectRes: StartRoundOperationResult{Error: errors.New("operation function is nil")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "start_nil", "nil_function").Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetric != nil {
				tt.mockMetric()
			}
			got, err := wrapStartRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, mockMetrics)

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
			} else if got.Error != nil {
				t.Errorf("Expected no error in result struct, got %v", got.Error)
			}
		})
	}
}
