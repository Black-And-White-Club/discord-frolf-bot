package createround

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewCreateRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fakeHelper := &testutils.FakeHelpers{}
	cfg := &config.Config{}
	fakeInteractionStore := testutils.NewFakeStorage[any]()
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

	manager := NewCreateRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, cfg, fakeInteractionStore, nil, tracer, fakeMetrics, fakeGuildConfigResolver)
	impl, ok := manager.(*createRoundManager)
	if !ok {
		t.Fatalf("Expected *createRoundManager, got %T", manager)
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
	if impl.config != cfg {
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

func Test_wrapCreateRoundOperation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (CreateRoundOperationResult, error)
		expectErr string
		expectRes CreateRoundOperationResult
		setup     func()
	}{
		{
			name:      "success path",
			operation: "create_success",
			fn: func(ctx context.Context) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: CreateRoundOperationResult{Success: "yay!"},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "create_success" {
						t.Errorf("expected operation create_success, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIRequestFunc = func(ctx context.Context, operation string) {
					if operation != "create_success" {
						t.Errorf("expected operation create_success, got %s", operation)
					}
				}
			},
		},
		{
			name:      "fn returns error",
			operation: "create_error",
			fn: func(ctx context.Context) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "create_error operation error: bad op",
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "create_error" {
						t.Errorf("expected operation create_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "create_error" || errorType != "operation_error" {
						t.Errorf("expected operation create_error and errorType operation_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "result has error",
			operation: "create_result_error",
			fn: func(ctx context.Context) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: CreateRoundOperationResult{Error: errors.New("oopsie")},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "create_result_error" {
						t.Errorf("expected operation create_result_error, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "create_result_error" || errorType != "result_error" {
						t.Errorf("expected operation create_result_error and errorType result_error, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "panic recovery",
			operation: "create_panic",
			fn: func(ctx context.Context) (CreateRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "",
			expectRes: CreateRoundOperationResult{Error: nil},
			setup: func() {
				fakeMetrics.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {
					if operation != "create_panic" {
						t.Errorf("expected operation create_panic, got %s", operation)
					}
				}
				fakeMetrics.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {
					if operation != "create_panic" || errorType != "panic" {
						t.Errorf("expected operation create_panic and errorType panic, got %s, %s", operation, errorType)
					}
				}
			},
		},
		{
			name:      "nil fn",
			operation: "create_nil",
			fn:        nil,
			expectRes: CreateRoundOperationResult{Error: errors.New("operation function is nil")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			got, err := wrapCreateRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
