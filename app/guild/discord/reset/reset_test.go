package reset

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewResetManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager, err := NewResetManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)
	if err != nil {
		t.Fatalf("NewResetManager() unexpected error: %v", err)
	}

	impl, ok := manager.(*resetManager)
	if !ok {
		t.Fatalf("Expected *resetManager, got %T", manager)
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
}

func TestNewResetManager_ValidatesSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager, err := NewResetManager(nil, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)

	if err == nil {
		t.Fatal("NewResetManager should error when session is nil")
	}
	if manager != nil {
		t.Fatal("NewResetManager should return nil manager when session is nil")
	}
	if !strings.Contains(err.Error(), "session cannot be nil") {
		t.Errorf("Expected error about session, got: %v", err)
	}
}

func TestNewResetManager_ValidatesPublisher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager, err := NewResetManager(mockSession, nil, logger, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)

	if err == nil {
		t.Fatal("NewResetManager should error when publisher is nil")
	}
	if manager != nil {
		t.Fatal("NewResetManager should return nil manager when publisher is nil")
	}
	if !strings.Contains(err.Error(), "publisher cannot be nil") {
		t.Errorf("Expected error about publisher, got: %v", err)
	}
}

func TestNewResetManager_ValidatesLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager, err := NewResetManager(mockSession, mockEventBus, nil, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)

	if err == nil {
		t.Fatal("NewResetManager should error when logger is nil")
	}
	if manager != nil {
		t.Fatal("NewResetManager should return nil manager when logger is nil")
	}
	if !strings.Contains(err.Error(), "logger cannot be nil") {
		t.Errorf("Expected error about logger, got: %v", err)
	}
}

func TestNewResetManager_ValidatesHelper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	manager, err := NewResetManager(mockSession, mockEventBus, logger, nil, mockConfig, mockInteractionStore, tracer, mockMetrics)

	if err == nil {
		t.Fatal("NewResetManager should error when helper is nil")
	}
	if manager != nil {
		t.Fatal("NewResetManager should return nil manager when helper is nil")
	}
	if !strings.Contains(err.Error(), "helper cannot be nil") {
		t.Errorf("Expected error about helper, got: %v", err)
	}
}

func Test_operationWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name      string
		operation string
		fn        func(context.Context) error
		expectErr string
	}{
		{
			name:      "success path",
			operation: "test_success",
			fn: func(ctx context.Context) error {
				return nil
			},
			expectErr: "",
		},
		{
			name:      "error path",
			operation: "test_error",
			fn: func(ctx context.Context) error {
				return errors.New("test error")
			},
			expectErr: "test_error failed: test error",
		},
		{
			name:      "context cancelled",
			operation: "test_cancelled",
			fn: func(ctx context.Context) error {
				return context.Canceled
			},
			expectErr: "test_cancelled failed: context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &resetManager{
				logger:  logger,
				tracer:  tracer,
				metrics: mockMetrics,
			}

			err := rm.operationWrapper(context.Background(), tt.operation, tt.fn)

			if tt.expectErr == "" && err != nil {
				t.Errorf("operationWrapper() unexpected error = %v", err)
			}
			if tt.expectErr != "" {
				if err == nil {
					t.Errorf("operationWrapper() expected error %q, got nil", tt.expectErr)
				} else if !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("operationWrapper() error = %q, want to contain %q", err.Error(), tt.expectErr)
				}
			}
		})
	}
}
