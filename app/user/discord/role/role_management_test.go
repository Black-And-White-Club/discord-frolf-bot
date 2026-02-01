package role

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test_roleManager_EditRoleUpdateResponse(t *testing.T) {
	// Test cases updated to match implementation
	fakeSession := &discord.FakeSession{}
	fakeInteractionStore := &FakeISInterface[any]{}

	tests := []struct {
		name           string
		setup          func()
		ctx            context.Context
		correlationID  string
		content        string
		wantErr        bool
		expectedResult RoleOperationResult
	}{
		{
			name: "successful role update response edit",
			setup: func() {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					if key == "correlation-id" {
						return &discordgo.Interaction{ID: "interaction-id", Token: "test-token"}, nil
					}
					return nil, errors.New("not found")
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, webhook *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					if interaction.ID != "interaction-id" {
						return nil, fmt.Errorf("unexpected interaction ID: %s", interaction.ID)
					}
					if webhook.Content == nil || *webhook.Content != "Role updated successfully." {
						return nil, fmt.Errorf("unexpected content: %v", webhook.Content)
					}
					return &discordgo.Message{}, nil
				}
				fakeInteractionStore.DeleteFunc = func(ctx context.Context, key string) {
				}
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Success: "response updated"},
		},
		{
			name: "failed to get interaction from store",
			setup: func() {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("item not found or expired")
				}
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-not-found",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: fmt.Errorf("interaction not found for correlation ID: correlation-id-not-found")},
		},
		{
			name: "stored interaction is not of the expected type",
			setup: func() {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return "not an interaction object", nil
				}
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-wrong-type",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: fmt.Errorf("interaction is not of the expected type")},
		},
		{
			name: "failed to edit interaction response",
			setup: func() {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "interaction-id-fail", Token: "test-token"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, webhook *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("discord API error")
				}
			},
			ctx:            context.Background(),
			correlationID:  "correlation-id-fail-edit",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: fmt.Errorf("failed to send result: discord API error")},
		},
		{
			name: "cancelled_context",
			setup: func() {
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			correlationID:  "correlation-id-cancel",
			content:        "Role updated successfully.",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: context.Canceled},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Instantiate roleManager within the test case scope
			rm := &roleManager{
				session:          fakeSession,
				publisher:        &FakeEventBus{},
				logger:           loggerfrolfbot.NoOpLogger,
				helper:           &FakeHelpers{},
				config:           &config.Config{},
				interactionStore: fakeInteractionStore,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				operationWrapper: func(ctx context.Context, operationName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return fn(ctx)
				},
			}

			// Setup mocks specific to this test case
			if tt.setup != nil {
				tt.setup()
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
	fakeSession := &discord.FakeSession{}

	// Test cases updated to match implementation
	tests := []struct {
		name           string
		setup          func()
		guildID        string
		userID         sharedtypes.DiscordID
		roleID         string
		wantErr        bool
		expectedResult RoleOperationResult
	}{
		{
			name: "successful role addition",
			setup: func() {
				fakeSession.GuildMemberRoleAddFunc = func(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
					return nil
				}
				fakeSession.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"role-1", "other-role"}}, nil
				}
			},
			guildID:        "guild-1",
			userID:         sharedtypes.DiscordID("user-1"),
			roleID:         "role-1",
			wantErr:        false,
			expectedResult: RoleOperationResult{Success: "role added"},
		},
		{
			name: "failed to add role - API error",
			setup: func() {
				fakeSession.GuildMemberRoleAddFunc = func(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
					return errors.New("discord API error on add")
				}
			},
			guildID:        "guild-2",
			userID:         sharedtypes.DiscordID("user-2"),
			roleID:         "role-2",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: errors.New("discord API error on add")},
		},
		{
			name: "failed to fetch member after adding role",
			setup: func() {
				fakeSession.GuildMemberRoleAddFunc = func(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
					return nil
				}
				fakeSession.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return nil, errors.New("discord API error on fetch")
				}
			},
			guildID:        "guild-3",
			userID:         sharedtypes.DiscordID("user-3"),
			roleID:         "role-3",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: errors.New("discord API error on fetch")},
		},
		{
			name: "role check fails after adding role (role not found in member object)",
			setup: func() {
				fakeSession.GuildMemberRoleAddFunc = func(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
					return nil
				}
				fakeSession.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{User: &discordgo.User{ID: userID}, Roles: []string{"other-role"}}, nil
				}
			},
			guildID:        "guild-4",
			userID:         sharedtypes.DiscordID("user-4"),
			roleID:         "role-4",
			wantErr:        false,
			expectedResult: RoleOperationResult{Error: fmt.Errorf("role role-4 was not added to user user-4")},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Instantiate roleManager within the test case scope
			rm := &roleManager{
				session:          fakeSession,
				publisher:        &FakeEventBus{},
				logger:           loggerfrolfbot.NoOpLogger,
				helper:           &FakeHelpers{},
				config:           &config.Config{},
				interactionStore: &FakeISInterface[any]{},
				tracer:           noop.NewTracerProvider().Tracer("test"),
				operationWrapper: func(ctx context.Context, operationName string, fn func(ctx context.Context) (RoleOperationResult, error)) (RoleOperationResult, error) {
					return fn(ctx) // Ignore the wrapper for testing
				},
			}

			// Setup mocks specific to this test case
			if tt.setup != nil {
				tt.setup()
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
