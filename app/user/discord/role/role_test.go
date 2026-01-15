package role

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewRoleManager(t *testing.T) {
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
				mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
				mockMetrics := discordmetrics.NewMockDiscordMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")
				mockGuildConfig := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

				// Call the function being tested
				// Note: NewRoleManager now accepts a guildConfigCache param before tracer; pass nil for the cache in tests
				manager, err := NewRoleManager(mockSession, mockEventBus, logger, mockHelper, mockConfig, mockGuildConfig, mockInteractionStore, nil, tracer, mockMetrics)
				// Ensure manager is correctly created
				if err != nil {
					t.Fatalf("NewRoleManager returned error: %v", err)
				}
				if manager == nil {
					t.Fatalf("NewRoleManager returned nil")
				}

				// Access the concrete type
				roleManagerImpl, ok := manager.(*roleManager)
				if !ok {
					t.Fatalf("manager is not of type *roleManager")
				}

				// Check that all dependencies were correctly assigned
				if roleManagerImpl.session != mockSession {
					t.Errorf("Session not correctly assigned")
				}
				if roleManagerImpl.publisher != mockEventBus {
					t.Errorf("EventBus not correctly assigned")
				}
				if roleManagerImpl.logger != logger {
					t.Errorf("Logger not correctly assigned")
				}
				if roleManagerImpl.helper != mockHelper {
					t.Errorf("Helper not correctly assigned")
				}
				if roleManagerImpl.config != mockConfig {
					t.Errorf("Config not correctly assigned")
				}
				if roleManagerImpl.interactionStore != mockInteractionStore {
					t.Errorf("InteractionStore not correctly assigned")
				}
				if roleManagerImpl.guildConfigCache != nil {
					t.Errorf("GuildConfigCache should be nil in this test")
				}
				if roleManagerImpl.tracer != tracer {
					t.Errorf("Tracer not correctly assigned")
				}
				if roleManagerImpl.metrics != mockMetrics {
					t.Errorf("Metrics not correctly assigned")
				}

				// Ensure operationWrapper is correctly set
				if roleManagerImpl.operationWrapper == nil {
					t.Errorf("operationWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				// Call with nil dependencies
				// pass nil for the added guildConfigCache param as well
				manager, err := NewRoleManager(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				// Ensure manager is correctly created
				if err != nil {
					t.Fatalf("NewRoleManager returned error: %v", err)
				}
				if manager == nil {
					t.Fatalf("NewRoleManager returned nil")
				}

				// Access the concrete type
				roleManagerImpl, ok := manager.(*roleManager)
				if !ok {
					t.Fatalf("manager is not of type *roleManager")
				}

				// Check nil fields
				if roleManagerImpl.session != nil {
					t.Errorf("Session should be nil")
				}
				if roleManagerImpl.publisher != nil {
					t.Errorf("Publisher should be nil")
				}
				if roleManagerImpl.logger != nil {
					t.Errorf("Logger should be nil")
				}
				if roleManagerImpl.helper != nil {
					t.Errorf("Helper should be nil")
				}
				if roleManagerImpl.config != nil {
					t.Errorf("Config should be nil")
				}
				if roleManagerImpl.interactionStore != nil {
					t.Errorf("InteractionStore should be nil")
				}
				if roleManagerImpl.tracer != nil {
					t.Errorf("Tracer should be nil")
				}
				if roleManagerImpl.metrics != nil {
					t.Errorf("Metrics should be nil")
				}

				// Ensure operationWrapper is still set
				if roleManagerImpl.operationWrapper == nil {
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

func Test_wrapRoleOperation(t *testing.T) {
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
		fn         func(ctx context.Context) (RoleOperationResult, error)
		wantResult RoleOperationResult
		wantErr    error
		setupMocks func()
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (RoleOperationResult, error) {
				return RoleOperationResult{
					Success: "test_success",
				}, nil
			},
			wantResult: RoleOperationResult{
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
			fn: func(ctx context.Context) (RoleOperationResult, error) {
				return RoleOperationResult{}, errors.New("test_error")
			},
			wantResult: RoleOperationResult{},
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
			fn: func(ctx context.Context) (RoleOperationResult, error) {
				return RoleOperationResult{
					Error: errors.New("result_error"),
				}, nil
			},
			wantResult: RoleOperationResult{
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
			wantResult: RoleOperationResult{},
			wantErr:    errors.New("operation function is nil"),
			setupMocks: func() {
				// No metric calls expected
			},
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			fn: func(ctx context.Context) (RoleOperationResult, error) {
				panic("test_panic")
			},
			wantResult: RoleOperationResult{},
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

			gotResult, err := wrapRoleOperation(tt.ctx, tt.operation, tt.fn, logger, tracer, mockMetrics)

			// Check error condition
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("wrapRoleOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If there should be an error, check that the error message contains the expected text
			if err != nil && tt.wantErr != nil {
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("wrapRoleOperation() error message = %q, want to contain %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			// For successful operations, check the result values
			if !reflect.DeepEqual(gotResult.Success, tt.wantResult.Success) {
				t.Errorf("wrapRoleOperation() Success = %v, want %v", gotResult.Success, tt.wantResult.Success)
			}

			if !reflect.DeepEqual(gotResult.Failure, tt.wantResult.Failure) {
				t.Errorf("wrapRoleOperation() Failure = %v, want %v", gotResult.Failure, tt.wantResult.Failure)
			}

			// Check Error field if relevant
			if tt.wantResult.Error != nil && gotResult.Error == nil {
				t.Errorf("wrapRoleOperation() Error = nil, want error")
			} else if tt.wantResult.Error == nil && gotResult.Error != nil {
				t.Errorf("wrapRoleOperation() Error = %v, want nil", gotResult.Error)
			} else if tt.wantResult.Error != nil && gotResult.Error != nil {
				if !strings.Contains(gotResult.Error.Error(), tt.wantResult.Error.Error()) {
					t.Errorf("wrapRoleOperation() Error = %v, want to contain %v", gotResult.Error, tt.wantResult.Error)
				}
			}
		})
	}
}
