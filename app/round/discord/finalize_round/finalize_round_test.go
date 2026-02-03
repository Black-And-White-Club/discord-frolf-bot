package finalizeround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
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

func TestNewFinalizeRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeTracer := noop.NewTracerProvider().Tracer("test")
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}
	fakeInteractionStore := testutils.NewFakeStorage[any]()
	fakeGuildConfigCache := testutils.NewFakeStorage[storage.GuildConfig]()

	manager := NewFinalizeRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, fakeTracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*finalizeRoundManager)
	if !ok {
		t.Fatalf("Expected *finalizeRoundManager, got %T", manager)
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
	if impl.tracer != fakeTracer {
		t.Error("Expected tracer to be assigned")
	}
	if impl.metrics != fakeMetrics {
		t.Error("Expected metrics to be assigned")
	}
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
	if impl.interactionStore != fakeInteractionStore {
		t.Error("Expected interactionStore to be assigned")
	}
	if impl.guildConfigCache != fakeGuildConfigCache {
		t.Error("Expected guildConfigCache to be assigned")
	}
}

func Test_wrapFinalizeRoundOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (FinalizeRoundOperationResult, error)
		expectErr string
		expectRes FinalizeRoundOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "finalize_success",
			fn: func(ctx context.Context) (FinalizeRoundOperationResult, error) {
				return FinalizeRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: FinalizeRoundOperationResult{Success: "yay!"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "finalize_success" {
						t.Errorf("expected operation finalize_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "finalize_success" {
						t.Errorf("expected operation finalize_success, got %s", operation)
					}
				}
			},
		},
		{
			name:      "fn returns error",
			operation: "finalize_error",
			fn: func(ctx context.Context) (FinalizeRoundOperationResult, error) {
				return FinalizeRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "finalize_error operation error: bad op",
			expectRes: FinalizeRoundOperationResult{Error: fmt.Errorf("finalize_error operation error: bad op")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "finalize_error" {
						t.Errorf("expected operation finalize_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "finalize_error" || errorType != "operation_error" {
						t.Errorf("expected operation finalize_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "finalize_result_error",
			fn: func(ctx context.Context) (FinalizeRoundOperationResult, error) {
				return FinalizeRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: FinalizeRoundOperationResult{Error: errors.New("oopsie")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "finalize_result_error" {
						t.Errorf("expected operation finalize_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "finalize_result_error" || errorType != "result_error" {
						t.Errorf("expected operation finalize_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "panic recovery",
			operation: "finalize_panic",
			fn: func(ctx context.Context) (FinalizeRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "",                                       // The outer function returns nil error on panic recovery
			expectRes: FinalizeRoundOperationResult{Error: nil}, // The struct's Error field is nil on panic recovery before assignment
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "finalize_panic" {
						t.Errorf("expected operation finalize_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "finalize_panic" || errorType != "panic" {
						t.Errorf("expected operation finalize_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "nil fn",
			operation: "finalize_nil",
			fn:        nil,
			expectRes: FinalizeRoundOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got, err := wrapFinalizeRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
