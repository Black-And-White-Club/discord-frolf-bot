package roundrsvp

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

func TestNewRoundRsvpManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

	manager := NewRoundRsvpManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*roundRsvpManager)
	if !ok {
		t.Fatalf("Expected *roundRsvpManager, got %T", manager)
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
	if impl.operationWrapper == nil {
		t.Error("Expected operationWrapper to be set")
	}
}

func Test_wrapRoundRsvpOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (RoundRsvpOperationResult, error)
		expectErr string
		expectRes RoundRsvpOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "handle_success",
			fn: func(ctx context.Context) (RoundRsvpOperationResult, error) {
				return RoundRsvpOperationResult{Success: "success"}, nil
			},
			expectRes: RoundRsvpOperationResult{Success: "success"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_success" {
						t.Errorf("expected operation handle_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "handle_success" {
						t.Errorf("expected operation handle_success, got %s", operation)
					}
				}
			},
		},
		{
			name:      "fn returns error",
			operation: "handle_error",
			fn: func(ctx context.Context) (RoundRsvpOperationResult, error) {
				return RoundRsvpOperationResult{}, errors.New("operation failed")
			},
			expectErr: "handle_error operation error: operation failed",
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_error" {
						t.Errorf("expected operation handle_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_error" || errorType != "operation_error" {
						t.Errorf("expected operation handle_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "handle_result_error",
			fn: func(ctx context.Context) (RoundRsvpOperationResult, error) {
				return RoundRsvpOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: RoundRsvpOperationResult{Error: errors.New("result error")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_result_error" {
						t.Errorf("expected operation handle_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_result_error" || errorType != "result_error" {
						t.Errorf("expected operation handle_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "panic recovery",
			operation: "handle_panic",
			fn: func(ctx context.Context) (RoundRsvpOperationResult, error) {
				panic("unexpected panic")
			},
			expectErr: "",
			expectRes: RoundRsvpOperationResult{Error: nil},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "handle_panic" {
						t.Errorf("expected operation handle_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "handle_panic" || errorType != "panic" {
						t.Errorf("expected operation handle_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "nil fn",
			operation: "handle_nil",
			fn:        nil,
			expectRes: RoundRsvpOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got, err := wrapRoundRsvpOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
