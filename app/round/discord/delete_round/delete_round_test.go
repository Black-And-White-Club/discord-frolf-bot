package deleteround

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewDeleteRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeInteractionStore := testutils.NewFakeStorage[any]()
	fakeGuildConfigCache := testutils.NewFakeStorage[storage.GuildConfig]()
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

	manager := NewDeleteRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*deleteRoundManager)
	if !ok {
		t.Fatalf("Expected *deleteRoundManager, got %T", manager)
	}

	if impl.session != fakeSession {
		t.Error("Expected session to be assigned")
	}
	if impl.publisher != fakeEventBus {
		t.Error("Expected publisher to be assigned")
	}
	if impl.logger != logger {
		t.Error("Expected logger to be assigned")
	}
	if impl.helper != fakeHelper {
		t.Error("Expected helper to be assigned")
	}
	if impl.config != fakeConfig {
		t.Error("Expected config to be assigned")
	}
	if impl.interactionStore != fakeInteractionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.tracer != tracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != fakeMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.guildConfigCache != fakeGuildConfigCache {
		t.Error("Expected guildConfigCache to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
}

func Test_wrapDeleteRoundOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (DeleteRoundOperationResult, error)
		expectErr string
		expectRes DeleteRoundOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "delete_success",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: DeleteRoundOperationResult{Success: "yay!"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "delete_success" {
						t.Errorf("expected operation delete_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "delete_success" {
						t.Errorf("expected operation delete_success, got %s", operation)
					}
				}
			},
		},
		{
			name:      "fn returns error",
			operation: "delete_error",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "delete_error operation error: bad op",
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "delete_error" {
						t.Errorf("expected operation delete_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "delete_error" || errorType != "operation_error" {
						t.Errorf("expected operation delete_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "delete_result_error",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				return DeleteRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: DeleteRoundOperationResult{Error: errors.New("oopsie")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "delete_result_error" {
						t.Errorf("expected operation delete_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "delete_result_error" || errorType != "result_error" {
						t.Errorf("expected operation delete_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "panic recovery",
			operation: "delete_panic",
			fn: func(ctx context.Context) (DeleteRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "panic in delete_panic",
			expectRes: DeleteRoundOperationResult{Error: errors.New("panic in delete_panic")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "delete_panic" {
						t.Errorf("expected operation delete_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "delete_panic" || errorType != "panic" {
						t.Errorf("expected operation delete_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
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
			if tt.setup != nil {
				tt.setup()
			}
			got, err := wrapDeleteRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
