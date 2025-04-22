package role

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"

	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operation func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
	return operation(ctx)
}

func Test_roleManager_RespondToRoleRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"Admin":  "role-id-admin",
				"Member": "role-id-member",
			},
		},
	}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &discordmetrics.NoOpMetrics{}

	rm := &roleManager{
		session:          mockSession,
		publisher:        mockPublisher,
		logger:           logger,
		helper:           mockHelper,
		config:           mockConfig,
		interactionStore: mockInteractionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: testOperationWrapper,
	}

	tests := []struct {
		name             string
		setup            func()
		ctx              context.Context
		interactionID    string
		interactionToken string
		targetUserID     sharedtypes.DiscordID
		wantSuccess      string
		wantErrMsg       string
		wantErrIs        error
	}{
		{
			name: "successful role request response",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Eq(&discordgo.Interaction{ID: "interaction-id", Token: "interaction-token"}),
						gomock.Any(),
					).
					Return(nil).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			targetUserID:     "target-user-id",
			wantSuccess:      "role request response sent",
			wantErrMsg:       "",
			wantErrIs:        nil,
		},
		{
			name: "context cancelled before interaction",
			setup: func() {
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately
				return ctx
			}(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			targetUserID:     "target-user-id",
			wantSuccess:      "",
			wantErrMsg:       context.Canceled.Error(),
			wantErrIs:        context.Canceled,
		},
		{
			name: "failed to respond to role request (API error)",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Eq(&discordgo.Interaction{ID: "interaction-id", Token: "interaction-token"}),
						gomock.Any(),
					).
					Return(errors.New("discord API error")).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			targetUserID:     "target-user-id",
			wantSuccess:      "",
			wantErrMsg:       "failed to respond to role request: discord API error",
			wantErrIs:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			// Call the function under test. The wrapper just passes through fn(ctx).
			result, err := rm.RespondToRoleRequest(tt.ctx, tt.interactionID, tt.interactionToken, tt.targetUserID)
			if err != nil {
				t.Fatalf("RespondToRoleRequest() second return value error was non-nil: %v; expected nil with pass-through wrapper", err)
			}

			// 2. Check the RoleOperationResult fields (Success and Error)
			if result.Success != tt.wantSuccess {
				t.Errorf("RoleOperationResult.Success mismatch: got %q, want %q", result.Success, tt.wantSuccess)
			}

			// 3. Check the RoleOperationResult.Error field
			if tt.wantErrMsg != "" {
				// Expecting an error within the result
				if result.Error == nil {
					t.Errorf("RoleOperationResult.Error is nil, expected error containing %q", tt.wantErrMsg)
				} else {
					// Check if the error message contains the expected substring
					if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
						t.Errorf("RoleOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
					}
					// Optionally, check for a specific error type if tt.wantErrIs is set
					if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
						t.Errorf("RoleOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
					}
				}
			} else {
				// Expecting no error within the result
				if result.Error != nil {
					t.Errorf("RoleOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}

func Test_roleManager_RespondToRoleButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"Admin":  "role-id-admin",
				"Member": "role-id-member",
			},
		},
	}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &discordmetrics.NoOpMetrics{}

	rm := &roleManager{
		session:          mockSession,
		publisher:        mockPublisher,
		logger:           logger,
		helper:           mockHelper,
		config:           mockConfig,
		interactionStore: mockInteractionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: testOperationWrapper,
	}

	tests := []struct {
		name             string
		setup            func()
		ctx              context.Context
		interactionID    string
		interactionToken string
		requesterID      sharedtypes.DiscordID
		selectedRole     string
		targetUserID     sharedtypes.DiscordID
		wantSuccess      string
		wantErrMsg       string
		wantErrIs        error
	}{
		{
			name: "successful button press acknowledgement",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			requesterID:      "requester-id",
			selectedRole:     "Admin",
			targetUserID:     "target-user-id",
			wantSuccess:      "button press acknowledged",
			wantErrMsg:       "",
			wantErrIs:        nil,
		},
		{
			name: "failed to acknowledge button press (API error)",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Any(),
						gomock.Any(),
					).
					Return(errors.New("discord API error")).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			requesterID:      "requester-id",
			selectedRole:     "Member",
			targetUserID:     "target-user-id",
			wantSuccess:      "",
			wantErrMsg:       "failed to acknowledge role button press: discord API error",
			wantErrIs:        nil,
		},
		{
			name: "context cancelled before operation",
			setup: func() {
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately
				return ctx
			}(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			requesterID:      "requester-id",
			selectedRole:     "Admin",
			targetUserID:     "target-user-id",
			wantSuccess:      "",                       // No success message on context cancellation
			wantErrMsg:       context.Canceled.Error(), // Expected error message from context
			wantErrIs:        context.Canceled,         // Expected context.Canceled error type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := rm.RespondToRoleButtonPress(tt.ctx, tt.interactionID, tt.interactionToken, tt.requesterID, tt.selectedRole, tt.targetUserID)
			if err != nil {
				t.Fatalf("RespondToRoleButtonPress() second return value error was non-nil: %v; expected nil with pass-through wrapper", err)
			}

			// Check the RoleOperationResult.Success field
			if result.Success != tt.wantSuccess {
				t.Errorf("RoleOperationResult.Success mismatch: got %q, want %q", result.Success, tt.wantSuccess)
			}

			// Check the RoleOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("RoleOperationResult.Error is nil, expected error containing %q", tt.wantErrMsg)
				} else {
					// Check if the error message contains the expected substring
					if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
						t.Errorf("RoleOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
					}
					// Optionally, check for a specific error type if tt.wantErrIs is set
					if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
						t.Errorf("RoleOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
					}
				}
			} else {
				if result.Error != nil {
					t.Errorf("RoleOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}

func Test_roleManager_HandleRoleRequestCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"Admin": "123456789",
				"User":  "987654321",
			},
		},
	}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	metrics := &discordmetrics.NoOpMetrics{}

	tests := []struct {
		name    string
		setup   func()
		ctx     context.Context
		i       *discordgo.InteractionCreate
		wantErr bool
	}{
		{
			name:    "nil interaction",
			setup:   func() {},
			ctx:     context.Background(),
			i:       nil,
			wantErr: false,
		},
		{
			name:  "nil interaction interaction",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: nil,
			},
			wantErr: false,
		},
		{
			name:  "nil user",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					User: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "successful role request handling",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to respond to role request",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("respond to role request error")).
					Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           logger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: func(ctx context.Context, operationName string, operation func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return operation(ctx)
				},
			}

			rm.HandleRoleRequestCommand(tt.ctx, tt.i)
		})
	}
}

func Test_roleManager_HandleRoleButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"Admin": "123456789",
				"User":  "987654321",
			},
		},
	}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	metrics := &discordmetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name      string
		setup     func()
		ctx       context.Context
		i         *discordgo.InteractionCreate
		wantErr   bool
		tagNumber sharedtypes.TagNumber
	}{
		{
			name:    "nil interaction",
			setup:   func() {},
			ctx:     context.Background(),
			i:       nil,
			wantErr: false,
		},
		{
			name:  "nil interaction interaction",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: nil,
			},
			wantErr: false,
		},
		{
			name: "unexpected interaction data type",
			setup: func() {
				// Add expectation for InteractionRespond that may be called
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				// Add expectation for InteractionResponseEdit with the correct number of arguments (3)
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					AnyTimes()
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    "interaction-id",
					User:  &discordgo.User{ID: "user-id"},
					Token: "interaction-token",
					Data:  &discordgo.ApplicationCommandInteractionData{},
				},
			},
			wantErr: false,
		},
		{
			name: "no mentions in message",
			setup: func() {
				// Add expectations for Discord API calls
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					AnyTimes()
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    "interaction-id",
					User:  &discordgo.User{ID: "user-id"},
					Token: "interaction-token",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid role button press",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					AnyTimes()

				// mockPublisher.EXPECT().
				// 	Publish(gomock.Any(), gomock.Any()).
				// 	Return(nil).
				// 	Times(1) // Ensure the event is published exactly once

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1) // Ensure the interaction is stored
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in storing interaction reference in cache",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					AnyTimes()

				// Create a message with initialized metadata
				msg := message.NewMessage(watermill.NewUUID(), nil)

				// Add expectation for CreateNewMessage
				mockHelper.EXPECT().
					CreateNewMessage(gomock.Any(), gomock.Any()).
					Return(msg, nil).
					AnyTimes()

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("error storing interaction reference")).
					AnyTimes()

				mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-id"},
					},
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in publishing event to JetStream",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					AnyTimes()

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to publish event")).
					AnyTimes()
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in sending error response to user",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					AnyTimes()

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, errors.New("failed to edit response")).
					AnyTimes()

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           logger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				metrics:          metrics,
				tracer:           tracer,
				operationWrapper: func(ctx context.Context, operationName string, operation func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return operation(ctx)
				},
			}

			rm.HandleRoleButtonPress(tt.ctx, tt.i)
		})
	}
}

func Test_roleManager_HandleRoleCancelButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &discordmetrics.NoOpMetrics{}

	tests := []struct {
		name             string
		setup            func()
		ctx              context.Context
		interactionID    string
		interactionToken string
		wantErr          bool
	}{
		{
			name: "successful role cancel button handling",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "failed to respond to interaction",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("respond to interaction error")).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "failed to delete interaction from store",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "nil_interaction",
			setup: func() {
				// Do not expect any interactions
			},
			ctx:              context.Background(),
			interactionID:    "",
			interactionToken: "",
			wantErr:          false,
		},
		{
			name: "empty interaction ID",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(0)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(0)
			},
			ctx:              context.Background(),
			interactionID:    "",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "cancelled_context",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(0) // No response should be sent
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(0) // No deletion should occur
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Immediately cancel the context
				return ctx
			}(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           logger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: func(ctx context.Context, operationName string, operation func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return operation(ctx)
				},
			}

			var interaction *discordgo.InteractionCreate
			if tt.interactionID != "" {
				interaction = &discordgo.InteractionCreate{
					Interaction: &discordgo.Interaction{
						ID: tt.interactionID,
					},
				}
			}

			if interaction != nil {
				rm.HandleRoleCancelButton(tt.ctx, interaction)
			} else {
				rm.HandleRoleCancelButton(tt.ctx, nil)
			}
		})
	}
}
