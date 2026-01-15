package leaderboardupdated

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardUpdateManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")
	mockGuildConfig := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	manager := NewLeaderboardUpdateManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockGuildConfig, mockInteractionStore, nil, tracer, mockMetrics)
	impl, ok := manager.(*leaderboardUpdateManager)
	if !ok {
		t.Fatalf("Expected *leaderboardUpdateManager, got %T", manager)
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

func Test_wrapLeaderboardUpdateOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name       string
		operation  string
		fn         func(context.Context) (LeaderboardUpdateOperationResult, error)
		expectErr  string
		expectRes  LeaderboardUpdateOperationResult
		mockMetric func()
	}{
		{
			name:      "success path",
			operation: "leaderboard_success",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{Success: "success!"}, nil
			},
			expectRes: LeaderboardUpdateOperationResult{Success: "success!"},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "leaderboard_success", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "leaderboard_success").Times(1)
			},
		},
		{
			name:      "fn returns error",
			operation: "leaderboard_error",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{}, errors.New("operation failed")
			},
			expectErr: "leaderboard_error operation error: operation failed",
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "leaderboard_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "leaderboard_error", "operation_error").Times(1)
			},
		},
		{
			name:      "result has error",
			operation: "leaderboard_result_error",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: LeaderboardUpdateOperationResult{Error: errors.New("result error")},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "leaderboard_result_error", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "leaderboard_result_error", "result_error").Times(1)
			},
		},
		{
			name:      "panic recovery",
			operation: "leaderboard_panic",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				panic("critical error")
			},
			expectErr: "panic in leaderboard_panic: critical error",
			expectRes: LeaderboardUpdateOperationResult{
				Error: errors.New("panic in leaderboard_panic: critical error"),
			},
			mockMetric: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "leaderboard_panic", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "leaderboard_panic", "panic").Times(1)
			},
		},
		{
			name:      "nil function",
			operation: "leaderboard_nil_fn",
			fn:        nil,
			expectRes: LeaderboardUpdateOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetric != nil {
				tt.mockMetric()
			}

			got, err := wrapLeaderboardUpdateOperation(context.Background(), tt.operation, tt.fn, logger, tracer, mockMetrics)

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
			if tt.expectRes.Error != nil {
				if got.Error == nil {
					t.Error("Expected error in result, got nil")
				} else if !strings.Contains(got.Error.Error(), tt.expectRes.Error.Error()) {
					t.Errorf("Error = %v, want %v", got.Error, tt.expectRes.Error)
				}
			}
		})
	}
}
