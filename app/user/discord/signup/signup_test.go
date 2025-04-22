package signup

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewSignupManager(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates manager with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Create mock dependencies
				mockSession := discordmocks.NewMockSession(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockHelper := utilsmocks.NewMockHelpers(ctrl)
				mockConfig := &config.Config{}
				mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
				mockMetrics := discordmetrics.NewMockDiscordMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested
				manager, err := NewSignupManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockInteractionStore, tracer, mockMetrics)
				// Ensure manager is correctly created
				if err != nil {
					t.Fatalf("NewSignupManager returned error: %v", err)
				}
				if manager == nil {
					t.Fatalf("NewSignupManager returned nil")
				}

				// Access the concrete type
				signupManagerImpl, ok := manager.(*signupManager)
				if !ok {
					t.Fatalf("manager is not of type *signupManager")
				}

				// Check that all dependencies were correctly assigned
				if signupManagerImpl.session != mockSession {
					t.Errorf("Session not correctly assigned")
				}
				if signupManagerImpl.publisher != mockEventBus {
					t.Errorf("EventBus not correctly assigned")
				}
				if signupManagerImpl.logger != logger {
					t.Errorf("Logger not correctly assigned")
				}
				if signupManagerImpl.helper != mockHelper {
					t.Errorf("Helper not correctly assigned")
				}
				if signupManagerImpl.config != mockConfig {
					t.Errorf("Config not correctly assigned")
				}
				if signupManagerImpl.interactionStore != mockInteractionStore {
					t.Errorf("InteractionStore not correctly assigned")
				}
				if signupManagerImpl.tracer != tracer {
					t.Errorf("Tracer not correctly assigned")
				}
				if signupManagerImpl.metrics != mockMetrics {
					t.Errorf("Metrics not correctly assigned")
				}

				// Ensure operationWrapper is correctly set
				if signupManagerImpl.operationWrapper == nil {
					t.Errorf("operationWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				// Call with nil dependencies
				manager, err := NewSignupManager(nil, nil, nil, nil, nil, nil, nil, nil)
				// Ensure manager is correctly created
				if err != nil {
					t.Fatalf("NewSignupManager returned error: %v", err)
				}
				if manager == nil {
					t.Fatalf("NewSignupManager returned nil")
				}

				// Access the concrete type
				signupManagerImpl, ok := manager.(*signupManager)
				if !ok {
					t.Fatalf("manager is not of type *signupManager")
				}

				// Check nil fields
				if signupManagerImpl.session != nil {
					t.Errorf("Session should be nil")
				}
				if signupManagerImpl.publisher != nil {
					t.Errorf("Publisher should be nil")
				}
				if signupManagerImpl.logger != nil {
					t.Errorf("Logger should be nil")
				}
				if signupManagerImpl.helper != nil {
					t.Errorf("Helper should be nil")
				}
				if signupManagerImpl.config != nil {
					t.Errorf("Config should be nil")
				}
				if signupManagerImpl.interactionStore != nil {
					t.Errorf("InteractionStore should be nil")
				}
				if signupManagerImpl.tracer != nil {
					t.Errorf("Tracer should be nil")
				}
				if signupManagerImpl.metrics != nil {
					t.Errorf("Metrics should be nil")
				}

				// Ensure operationWrapper is still set
				if signupManagerImpl.operationWrapper == nil {
					t.Errorf("operationWrapper should not be nil")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_wrapSignupOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := discordmetrics.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name       string
		ctx        context.Context
		operation  string
		fn         func(ctx context.Context) (SignupOperationResult, error)
		wantResult SignupOperationResult
		wantErr    error
		setupMocks func()
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (SignupOperationResult, error) {
				return SignupOperationResult{
					Success: "test_success",
				}, nil
			},
			wantResult: SignupOperationResult{
				Success: "test_success",
			},
			wantErr: nil,
			setupMocks: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "test_operation", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIRequest(gomock.Any(), "test_operation").Times(1)
			},
		},
		{
			name:      "operation returns error",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (SignupOperationResult, error) {
				return SignupOperationResult{}, errors.New("test_error")
			},
			wantResult: SignupOperationResult{},
			wantErr:    errors.New("test_operation operation error: test_error"),
			setupMocks: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "test_operation", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "test_operation", "operation_error").Times(1)
			},
		},
		{
			name:      "operation result contains error",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (SignupOperationResult, error) {
				return SignupOperationResult{
					Error: errors.New("result_error"),
				}, nil
			},
			wantResult: SignupOperationResult{
				Error: errors.New("result_error"),
			},
			wantErr: nil,
			setupMocks: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "test_operation", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "test_operation", "result_error").Times(1)
			},
		},
		{
			name:       "nil function",
			ctx:        context.Background(),
			operation:  "test_operation",
			fn:         nil,
			wantResult: SignupOperationResult{},
			wantErr:    errors.New("operation function is nil"),
			setupMocks: func() {
				// No metric calls expected
			},
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (SignupOperationResult, error) {
				panic("test_panic")
			},
			wantResult: SignupOperationResult{},
			wantErr:    errors.New("panic in test_operation"),
			setupMocks: func() {
				mockMetrics.EXPECT().RecordAPIRequestDuration(gomock.Any(), "test_operation", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordAPIError(gomock.Any(), "test_operation", "panic").Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup any mock expectations
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			gotResult, err := wrapSignupOperation(tt.ctx, tt.operation, tt.fn, logger, tracer, mockMetrics)

			// Check error condition
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("wrapSignupOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If there should be an error, check that the error message contains the expected text
			if err != nil && tt.wantErr != nil {
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("wrapSignupOperation() error message = %q, want to contain %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			// For successful operations, check the result values
			if !reflect.DeepEqual(gotResult.Success, tt.wantResult.Success) {
				t.Errorf("wrapSignupOperation() Success = %v, want %v", gotResult.Success, tt.wantResult.Success)
			}

			if !reflect.DeepEqual(gotResult.Failure, tt.wantResult.Failure) {
				t.Errorf("wrapSignupOperation() Failure = %v, want %v", gotResult.Failure, tt.wantResult.Failure)
			}

			// Check Error field if relevant
			if tt.wantResult.Error != nil && gotResult.Error == nil {
				t.Errorf("wrapSignupOperation() Error = nil, want error")
			} else if tt.wantResult.Error == nil && gotResult.Error != nil {
				t.Errorf("wrapSignupOperation() Error = %v, want nil", gotResult.Error)
			} else if tt.wantResult.Error != nil && gotResult.Error != nil {
				if !strings.Contains(gotResult.Error.Error(), tt.wantResult.Error.Error()) {
					t.Errorf("wrapSignupOperation() Error = %v, want to contain %v", gotResult.Error, tt.wantResult.Error)
				}
			}
		})
	}
}

func Test_signupManager_createEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock dependencies
	mockSession := discordmocks.NewMockSession(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)

	// Create a simple config with just the required GuildID
	mockConfig := &config.Config{}
	mockConfig.Discord.GuildID = "test-guild-id"

	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockMetrics := discordmetrics.NewMockDiscordMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	// Create a signupManager instance
	sm := &signupManager{
		session:          mockSession,
		publisher:        mockEventBus,
		logger:           logger,
		helper:           mockHelper,
		config:           mockConfig,
		interactionStore: mockInteractionStore,
		tracer:           tracer,
		metrics:          mockMetrics,
	}

	// Create a mock interaction
	mockInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    "test-interaction-id",
			Token: "test-interaction-token",
		},
	}

	tests := []struct {
		name    string
		topic   string
		payload interface{}
		wantErr bool
	}{
		{
			name:    "successful event creation",
			topic:   "test_topic",
			payload: map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name:    "unmarshalable payload",
			topic:   "test_topic",
			payload: make(chan int), // Channels can't be marshaled to JSON
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := sm.createEvent(context.Background(), tt.topic, tt.payload, mockInteraction)

			if (err != nil) != tt.wantErr {
				t.Errorf("createEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify message fields
				if msg == nil {
					t.Errorf("createEvent() returned nil message")
					return
				}

				// Check metadata fields
				if msg.Metadata.Get("handler_name") != "Signup Event" {
					t.Errorf("Expected handler_name = %q, got %q", "Signup Event", msg.Metadata.Get("handler_name"))
				}

				if msg.Metadata.Get("topic") != tt.topic {
					t.Errorf("Expected topic = %q, got %q", tt.topic, msg.Metadata.Get("topic"))
				}

				if msg.Metadata.Get("domain") != "discord" {
					t.Errorf("Expected domain = %q, got %q", "discord", msg.Metadata.Get("domain"))
				}

				if msg.Metadata.Get("interaction_id") != mockInteraction.Interaction.ID {
					t.Errorf("Expected interaction_id = %q, got %q", mockInteraction.Interaction.ID, msg.Metadata.Get("interaction_id"))
				}

				if msg.Metadata.Get("interaction_token") != mockInteraction.Interaction.Token {
					t.Errorf("Expected interaction_token = %q, got %q", mockInteraction.Interaction.Token, msg.Metadata.Get("interaction_token"))
				}

				if msg.Metadata.Get("guild_id") != mockConfig.Discord.GuildID {
					t.Errorf("Expected guild_id = %q, got %q", mockConfig.Discord.GuildID, msg.Metadata.Get("guild_id"))
				}
			}
		})
	}
}
