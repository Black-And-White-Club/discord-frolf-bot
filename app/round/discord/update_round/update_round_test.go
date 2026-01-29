package updateround

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

func TestNewUpdateRoundManager(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	fakeHelper := &testutils.FakeHelpers{}
	mockConfig := &config.Config{}
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	fakeGuildConfigCache := &testutils.FakeStorage[storage.GuildConfig]{}
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	fakeGuildConfig := &testutils.FakeGuildConfigResolver{}

	manager := NewUpdateRoundManager(fakeSession, fakeEventBus, logger, fakeHelper, mockConfig, fakeInteractionStore, fakeGuildConfigCache, tracer, fakeMetrics, fakeGuildConfig)
	impl, ok := manager.(*updateRoundManager)
	if !ok {
		t.Fatalf("Expected *updateRoundManager, got %T", manager)
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
	if impl.config != mockConfig {
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

func Test_wrapUpdateRoundOperation(t *testing.T) {
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) (UpdateRoundOperationResult, error)
		expectErr string
		expectRes UpdateRoundOperationResult
		setupFake func(fm *testutils.FakeDiscordMetrics)
	}{
		{
			name:      "success path",
			operation: "update_success",
			fn: func(ctx context.Context) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{Success: "yay!"}, nil
			},
			expectRes: UpdateRoundOperationResult{Success: "yay!"},
			setupFake: func(fm *testutils.FakeDiscordMetrics) {
				fm.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {}
				fm.RecordAPIRequestFunc = func(ctx context.Context, operation string) {}
			},
		},
		{
			name:      "fn returns error",
			operation: "update_error",
			fn: func(ctx context.Context) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{}, errors.New("bad op")
			},
			expectErr: "update_error operation error: bad op",
			setupFake: func(fm *testutils.FakeDiscordMetrics) {
				fm.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {}
				fm.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {}
			},
		},
		{
			name:      "result has error",
			operation: "update_result_error",
			fn: func(ctx context.Context) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{Error: errors.New("oopsie")}, nil
			},
			expectRes: UpdateRoundOperationResult{Error: errors.New("oopsie")},
			setupFake: func(fm *testutils.FakeDiscordMetrics) {
				fm.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {}
				fm.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {}
			},
		},
		{
			name:      "panic recovery",
			operation: "update_panic",
			fn: func(ctx context.Context) (UpdateRoundOperationResult, error) {
				panic("oh no")
			},
			expectErr: "",
			expectRes: UpdateRoundOperationResult{Error: nil},
			setupFake: func(fm *testutils.FakeDiscordMetrics) {
				fm.RecordAPIRequestDurationFunc = func(ctx context.Context, operation string, duration time.Duration) {}
				fm.RecordAPIErrorFunc = func(ctx context.Context, operation, errorType string) {}
			},
		},
		{
			name:      "nil fn",
			operation: "update_nil",
			fn:        nil,
			expectRes: UpdateRoundOperationResult{Error: errors.New("operation function is nil")},
			setupFake: func(fm *testutils.FakeDiscordMetrics) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeMetrics := &testutils.FakeDiscordMetrics{}
			if tt.setupFake != nil {
				tt.setupFake(fakeMetrics)
			}
			got, err := wrapUpdateRoundOperation(context.Background(), tt.operation, tt.fn, logger, tracer, fakeMetrics)

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
