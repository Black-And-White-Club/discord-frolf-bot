package startround

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

func TestNewStartRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}

	fakeTracer := noop.NewTracerProvider().Tracer("test")
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	fakeGuildConfig := &testutils.FakeGuildConfigResolver{}

	manager := NewStartRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, fakeConfig, fakeInteractionStore, fakeGuildConfigCache, fakeTracer, fakeMetrics, fakeGuildConfig)
	impl, ok := manager.(*startRoundManager)
	if !ok {
		t.Fatalf("Expected *startRoundManager, got %T", manager)
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

func Test_wrapStartRoundOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeMetrics := &testutils.FakeDiscordMetrics{}
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
				var durCalled, reqCalled bool
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, op string, d time.Duration) {
					if op == "start_success" {
						durCalled = true
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, op string) {
					if op == "start_success" {
						reqCalled = true
					}
				}
				t.Cleanup(func() {
					if !durCalled {
						t.Errorf("expected RecordAPIRequestDuration called for start_success")
					}
					if !reqCalled {
						t.Errorf("expected RecordAPIRequest called for start_success")
					}
				})
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
				var durCalled, errCalled bool
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, op string, d time.Duration) {
					if op == "start_error" {
						durCalled = true
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, op, errType string) {
					if op == "start_error" && errType == "operation_error" {
						errCalled = true
					}
				}
				t.Cleanup(func() {
					if !durCalled {
						t.Errorf("expected RecordAPIRequestDuration called for start_error")
					}
					if !errCalled {
						t.Errorf("expected RecordAPIError called for start_error")
					}
				})
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
				var durCalled, errCalled bool
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, op string, d time.Duration) {
					if op == "start_result_error" {
						durCalled = true
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, op, errType string) {
					if op == "start_result_error" && errType == "result_error" {
						errCalled = true
					}
				}
				t.Cleanup(func() {
					if !durCalled {
						t.Errorf("expected RecordAPIRequestDuration called for start_result_error")
					}
					if !errCalled {
						t.Errorf("expected RecordAPIError called for start_result_error")
					}
				})
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
				var durCalled, errCalled bool
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, op string, d time.Duration) {
					if op == "start_panic" {
						durCalled = true
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, op, errType string) {
					if op == "start_panic" && errType == "panic" {
						errCalled = true
					}
				}
				t.Cleanup(func() {
					if !durCalled {
						t.Errorf("expected RecordAPIRequestDuration called for start_panic")
					}
					if !errCalled {
						t.Errorf("expected RecordAPIError called for start_panic")
					}
				})
			},
		},
		{
			name:      "nil fn",
			operation: "start_nil",
			fn:        nil,
			expectErr: "operation function is nil",
			expectRes: StartRoundOperationResult{Error: errors.New("operation function is nil")},
			mockMetric: func() {
				var errCalled bool
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, op, errType string) {
					if op == "start_nil" && errType == "nil_function" {
						errCalled = true
					}
				}
				t.Cleanup(func() {
					if !errCalled {
						t.Errorf("expected RecordAPIError called for start_nil")
					}
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockMetric != nil {
				tt.mockMetric()
			}
			got, err := wrapStartRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
