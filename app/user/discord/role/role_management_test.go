package role

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_roleManager_EditRoleUpdateResponse(t *testing.T) {
	// Test cases updated to match implementation
	tests := []struct {
		name           string
		setup          func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface)
		ctx            context.Context
		correlationID  string
		content        string
		wantErr        bool
		expectedResult RoleOperationResult
	}{
		{
			name: "successful role update response edit",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(&discordgo.Interaction{ID: "interaction-id", Token: "test-token"}, true).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					DoAndReturn(func(interaction *discordgo.Interaction, webhook *discordgo.WebhookEdit, options ...any) (*discordgo.Message, error) {
						if interaction.ID != "interaction-id" {
							return nil, fmt.Errorf("unexpected interaction ID: %s", interaction.ID)
						}
						if webhook.Content == nil || *webhook.Content != "Role updated successfully." {
							return nil, fmt.Errorf("unexpected content: %v", webhook.Content)
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Success: "response updated"},
		},
		{
			name: "failed to get interaction from store",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Get("correlation-id-not-found").
					Return(nil, false).
					Times(1)
				// No call to InteractionResponseEdit expected
				mockSession.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Times(0)
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-not-found",
			content:        "Role updated successfully.",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: fmt.Errorf("interaction not found for correlation ID: correlation-id-not-found")},
		},
		{
			name: "stored interaction is not of the expected type",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Get("correlation-id-wrong-type").
					Return("not an interaction object", true).
					Times(1)
				// No call to InteractionResponseEdit expected
				mockSession.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Times(0)
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-wrong-type",
			content:        "Role updated successfully.",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: fmt.Errorf("interaction is not of the expected type")},
		},
		{
			name: "failed to edit interaction response",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Get("correlation-id-fail-edit").
					Return(&discordgo.Interaction{ID: "interaction-id-fail", Token: "test-token"}, true).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("discord API error")).
					Times(1)
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-fail-edit",
			content:        "Role updated successfully.",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: fmt.Errorf("failed to send result: discord API error")},
		},
		{
			name: "cancelled_context",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				// No mocks expected as the context cancellation should prevent further calls
				mockInteractionStore.EXPECT().Get(gomock.Any()).Times(0)
				mockSession.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Times(0)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			correlationID:  "correlation-id-cancel",
			content:        "Role updated successfully.",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: context.Canceled},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mocks within the test case scope
			mockSession := discordmocks.NewMockSession(ctrl)
			mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockConfig := &config.Config{}
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
			mockHelper := util_mocks.NewMockHelpers(ctrl)
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test-tracer")

			// Instantiate roleManager within the test case scope
			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				operationWrapper: func(ctx context.Context, operationName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return fn(ctx)
				},
			}

			// Setup mocks specific to this test case
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore) // Pass mocks to setup func
			}

			// Execute the method under test
			result, err := rm.EditRoleUpdateResponse(tt.ctx, tt.correlationID, tt.content)

			// Assertions
			if (err != nil) != tt.wantErr {
				t.Errorf("roleManager.EditRoleUpdateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Compare the error in result separately since Error objects can't be directly compared
			if tt.expectedResult.Error != nil && result.Error == nil {
				t.Errorf("roleManager.EditRoleUpdateResponse() expected error in result, got nil")
			} else if tt.expectedResult.Error == nil && result.Error != nil {
				t.Errorf("roleManager.EditRoleUpdateResponse() unexpected error in result: %v", result.Error)
			} else if tt.expectedResult.Error != nil && result.Error != nil {
				// Check if error messages match
				if tt.expectedResult.Error.Error() != result.Error.Error() {
					t.Errorf("roleManager.EditRoleUpdateResponse() error message mismatch: expected %v, got %v",
						tt.expectedResult.Error, result.Error)
				}
			}

			// Compare the success field
			if tt.expectedResult.Success != result.Success {
				t.Errorf("roleManager.EditRoleUpdateResponse() result success = %v, want %v",
					result.Success, tt.expectedResult.Success)
			}
		})
	}
}

func Test_roleManager_AddRoleToUser(t *testing.T) {
	// Test cases updated to match implementation
	tests := []struct {
		name           string
		setup          func(mockSession *discordmocks.MockSession)
		guildID        string
		userID         sharedtypes.DiscordID
		roleID         string
		wantErr        bool
		expectedResult RoleOperationResult
	}{
		{
			name: "successful role addition",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-1", "user-1", "role-1").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-1", "user-1").
					Return(&discordgo.Member{User: &discordgo.User{ID: "user-1"}, Roles: []string{"role-1", "other-role"}}, nil).
					Times(1)
			},
			guildID:        "guild-1",
			userID:         sharedtypes.DiscordID("user-1"),
			roleID:         "role-1",
			wantErr:        false,
			expectedResult: RoleOperationResult{Success: "role added"},
		},
		{
			name: "failed to add role - API error",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-2", "user-2", "role-2").
					Return(errors.New("discord API error on add")).
					Times(1)
				mockSession.EXPECT().GuildMember(gomock.Any(), gomock.Any()).Times(0)
			},
			guildID:        "guild-2",
			userID:         sharedtypes.DiscordID("user-2"),
			roleID:         "role-2",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: errors.New("discord API error on add")},
		},
		{
			name: "failed to fetch member after adding role",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-3", "user-3", "role-3").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-3", "user-3").
					Return(nil, errors.New("discord API error on fetch")).
					Times(1)
			},
			guildID:        "guild-3",
			userID:         sharedtypes.DiscordID("user-3"),
			roleID:         "role-3",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: errors.New("discord API error on fetch")},
		},
		{
			name: "role check fails after adding role (role not found in member object)",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-4", "user-4", "role-4").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-4", "user-4").
					Return(&discordgo.Member{User: &discordgo.User{ID: "user-4"}, Roles: []string{"other-role"}}, nil).
					Times(1)
			},
			guildID:        "guild-4",
			userID:         sharedtypes.DiscordID("user-4"),
			roleID:         "role-4",
			wantErr:        false, // Changed from true because error is in result object now
			expectedResult: RoleOperationResult{Error: fmt.Errorf("role role-4 was not added to user user-4")},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mocks within the test case scope
			mockSession := discordmocks.NewMockSession(ctrl)
			mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockConfig := &config.Config{}
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
			mockHelper := util_mocks.NewMockHelpers(ctrl)
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test-tracer")

			// Instantiate roleManager within the test case scope
			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				operationWrapper: func(ctx context.Context, operationName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return fn(ctx) // Ignore the wrapper for testing
				},
			}

			// Setup mocks specific to this test case
			if tt.setup != nil {
				tt.setup(mockSession)
			}

			// Execute the method under test
			result, err := rm.AddRoleToUser(context.Background(), tt.guildID, tt.userID, tt.roleID)

			// Assertions
			if (err != nil) != tt.wantErr {
				t.Errorf("roleManager.AddRoleToUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Compare the error in result separately since Error objects can't be directly compared
			if tt.expectedResult.Error != nil && result.Error == nil {
				t.Errorf("roleManager.AddRoleToUser() expected error in result, got nil")
			} else if tt.expectedResult.Error == nil && result.Error != nil {
				t.Errorf("roleManager.AddRoleToUser() unexpected error in result: %v", result.Error)
			} else if tt.expectedResult.Error != nil && result.Error != nil {
				// Check if error messages match
				if tt.expectedResult.Error.Error() != result.Error.Error() {
					t.Errorf("roleManager.AddRoleToUser() error message mismatch: expected %v, got %v",
						tt.expectedResult.Error, result.Error)
				}
			}

			// Compare the success field
			if tt.expectedResult.Success != result.Success {
				t.Errorf("roleManager.AddRoleToUser() result success = %v, want %v",
					result.Success, tt.expectedResult.Success)
			}
		})
	}
}
