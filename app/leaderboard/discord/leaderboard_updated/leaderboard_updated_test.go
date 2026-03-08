package leaderboardupdated

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewLeaderboardUpdateManager(t *testing.T) {
	// Use FakeSession instead of gomock
	fakeSession := discord.NewFakeSession()

	// Use nil for dependencies since no methods are called during construction
	var publisher eventbus.EventBus = nil
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	var helper utils.Helpers = nil
	mockConfig := &config.Config{}
	var interactionStore storage.ISInterface[any] = nil
	var metrics discordmetrics.DiscordMetrics = nil
	tracer := noop.NewTracerProvider().Tracer("test")
	var guildConfigResolver guildconfig.GuildConfigResolver = nil

	manager := NewLeaderboardUpdateManager(fakeSession, publisher, logger, helper, mockConfig, guildConfigResolver, interactionStore, nil, tracer, metrics)
	impl, ok := manager.(*leaderboardUpdateManager)
	if !ok {
		t.Fatalf("Expected *leaderboardUpdateManager, got %T", manager)
	}

	if impl.session != fakeSession {
		t.Error("Expected session to be assigned")
	}
	if impl.publisher != publisher {
		t.Error("Expected publisher to be assigned")
	}
	if impl.logger != logger {
		t.Error("Expected logger to be assigned")
	}
	if impl.helper != helper {
		t.Error("Expected helper to be assigned")
	}
	if impl.config != mockConfig {
		t.Error("Expected config to be assigned")
	}
	if impl.interactionStore != interactionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.tracer != tracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != metrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
}

func Test_wrapLeaderboardUpdateOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (LeaderboardUpdateOperationResult, error)
		expectErr string
		expectRes LeaderboardUpdateOperationResult
	}{
		{
			name:      "success path",
			operation: "leaderboard_success",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{Success: "success!"}, nil
			},
			expectRes: LeaderboardUpdateOperationResult{Success: "success!"},
		},
		{
			name:      "fn returns error",
			operation: "leaderboard_error",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{}, errors.New("operation failed")
			},
			expectErr: "leaderboard_error operation error: operation failed",
		},
		{
			name:      "result has error",
			operation: "leaderboard_result_error",
			fn: func(ctx context.Context) (LeaderboardUpdateOperationResult, error) {
				return LeaderboardUpdateOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: LeaderboardUpdateOperationResult{Error: errors.New("result error")},
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
			// Use a fake metrics implementation
			fakeMetrics := &fakeDiscordMetrics{}

			got, err := wrapLeaderboardUpdateOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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

// fakeDiscordMetrics is a no-op implementation of DiscordMetrics for testing.
type fakeDiscordMetrics struct{}

func (f *fakeDiscordMetrics) RecordAPIRequestDuration(ctx context.Context, endpoint string, duration time.Duration) {
}
func (f *fakeDiscordMetrics) RecordAPIRequest(ctx context.Context, endpoint string)                 {}
func (f *fakeDiscordMetrics) RecordAPIError(ctx context.Context, endpoint string, errorType string) {}
func (f *fakeDiscordMetrics) RecordRateLimit(ctx context.Context, endpoint string, resetTime time.Duration) {
}
func (f *fakeDiscordMetrics) RecordWebsocketEvent(ctx context.Context, eventType string)   {}
func (f *fakeDiscordMetrics) RecordWebsocketReconnect(ctx context.Context)                 {}
func (f *fakeDiscordMetrics) RecordWebsocketDisconnect(ctx context.Context, reason string) {}
func (f *fakeDiscordMetrics) RecordHandlerAttempt(ctx context.Context, handlerName string) {}
func (f *fakeDiscordMetrics) RecordHandlerSuccess(ctx context.Context, handlerName string) {}
func (f *fakeDiscordMetrics) RecordHandlerFailure(ctx context.Context, handlerName string) {}
func (f *fakeDiscordMetrics) RecordHandlerDuration(ctx context.Context, handlerName string, duration time.Duration) {
}
