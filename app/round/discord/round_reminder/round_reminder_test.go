package roundreminder

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
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewRoundReminderManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeTracer := noop.NewTracerProvider().Tracer("test")
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

	var nilStoreAny storage.ISInterface[any] = nil
	var nilStoreGuild storage.ISInterface[storage.GuildConfig] = nil
	manager := NewRoundReminderManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, nilStoreAny, nilStoreGuild, fakeTracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*roundReminderManager)
	if !ok {
		t.Fatalf("Expected *roundReminderManager, got %T", manager)
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
}

func TestNewRoundReminderManager_WithNilLogger(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeTracer := noop.NewTracerProvider().Tracer("test")
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

	// Test with nil logger
	var nilStoreAny storage.ISInterface[any] = nil
	var nilStoreGuild storage.ISInterface[storage.GuildConfig] = nil
	manager := NewRoundReminderManager(fakeSession, fakeEventBus, nil, fakeHelper, fakeConfig, nilStoreAny, nilStoreGuild, fakeTracer, fakeMetrics, fakeGuildConfigResolver)
	if manager == nil {
		t.Fatal("Expected manager to be created even with nil logger")
	}
}

func Test_wrapRoundReminderOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (RoundReminderOperationResult, error)
		expectErr string
		expectRes RoundReminderOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "reminder_success",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Success: "success"}, nil
			},
			expectRes: RoundReminderOperationResult{Success: "success"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "reminder_success" {
						t.Errorf("expected operation reminder_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "reminder_success" {
						t.Errorf("expected operation reminder_success, got %s", operation)
					}
				}
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
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "reminder_error" {
						t.Errorf("expected operation reminder_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "reminder_error" || errorType != "operation_error" {
						t.Errorf("expected operation reminder_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "reminder_result_error",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Error: errors.New("result error")}, nil
			},
			expectRes: RoundReminderOperationResult{Error: errors.New("result error")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "reminder_result_error" {
						t.Errorf("expected operation reminder_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "reminder_result_error" || errorType != "result_error" {
						t.Errorf("expected operation reminder_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
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
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "reminder_panic" {
						t.Errorf("expected operation reminder_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "reminder_panic" || errorType != "panic" {
						t.Errorf("expected operation reminder_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
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
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "reminder_nil_tracer" {
						t.Errorf("expected operation reminder_nil_tracer, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "reminder_nil_tracer" {
						t.Errorf("expected operation reminder_nil_tracer, got %s", operation)
					}
				}
			},
		},
		{
			name:      "nil metrics",
			operation: "reminder_nil_metrics",
			fn: func(ctx context.Context) (RoundReminderOperationResult, error) {
				return RoundReminderOperationResult{Success: "success with nil metrics"}, nil
			},
			expectRes: RoundReminderOperationResult{Success: "success with nil metrics"},
			// No setup for nil metrics case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metricsToUse discordmetrics.DiscordMetrics
			if tt.name == "nil metrics" {
				metricsToUse = nil
			} else {
				metricsToUse = fakeMetrics
				if tt.setup != nil {
					tt.setup()
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
