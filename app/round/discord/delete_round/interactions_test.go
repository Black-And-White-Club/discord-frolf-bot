package deleteround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	helpersmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_deleteRoundManager_HandleDeleteRoundCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	metrics := &discordmetrics.NoOpMetrics{}
	logger := loggerfrolfbot.NoOpLogger

	// Helper function to create a sample InteractionCreate with the desired custom ID
	createInteraction := func(customID string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-456",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-456",
						Username: "TestUser",
					},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID:      customID,
					ComponentType: discordgo.ButtonComponent,
				},
				Type: discordgo.InteractionMessageComponent,
			},
		}
	}

	// Helper function to create an expected round delete request payload
	createExpectedPayload := func(roundID sharedtypes.RoundID) roundevents.RoundDeleteRequestPayload {
		return roundevents.RoundDeleteRequestPayload{
			RoundID:              roundID,
			RequestingUserUserID: sharedtypes.DiscordID("user-456"),
		}
	}

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
		expectErr     bool
		expectedError string
		expectSuccess bool
	}{
		{
			name: "successful delete request",
			setup: func() {
				// Mock InteractionRespond call
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				roundUUID := uuid.MustParse("00000000-0000-0000-0000-0000000003b1") // Example UUID
				expectedPayload := createExpectedPayload(sharedtypes.RoundID(roundUUID))
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedPayload), gomock.Eq(roundevents.RoundDeleteRequest)).
					Return(&message.Message{UUID: "msg-456"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundDeleteRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any()).
					Return(&discordgo.Message{ID: "message-456"}, nil).
					Times(1)

				// mockLogger.EXPECT().InfoContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes() // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"), // Use valid UUID string
			},
			expectSuccess: true,
			expectErr:     false,
		},
		{
			name: "invalid custom ID format",
			setup: func() {
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete"), // Missing the round ID part
			},
			expectErr:     true,
			expectedError: "invalid custom_id format",
		},
		{
			name: "invalid round ID",
			setup: func() {
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any()).
					Return(&discordgo.Message{ID: "message-456"}, nil).
					Times(1)
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|invalid"),
			},
			expectErr:     true,
			expectedError: "failed to parse round ID as UUID",
		},
		{
			name: "interaction respond error",
			setup: func() {
				// Mock InteractionRespond call with error
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to acknowledge interaction",
		},
		{
			name: "create result message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundDeleteRequest)).
					Return(nil, errors.New("failed to create result message")).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any()).
					Return(&discordgo.Message{ID: "message-456"}, nil).
					Times(1)
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to create result message",
		},
		{
			name: "publish error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundDeleteRequest)).
					Return(&message.Message{UUID: "msg-456"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundDeleteRequest), gomock.Any()).
					Return(errors.New("failed to publish message")).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any()).
					Return(&discordgo.Message{ID: "message-456"}, nil).
					Times(1)
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to publish delete request",
		},
		{
			name: "followup message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundDeleteRequest)).
					Return(&message.Message{UUID: "msg-456"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundDeleteRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any()).
					Return(nil, errors.New("failed to send followup message")).
					Times(1)
				// mockLogger.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1) // Remove this
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to send followup message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			drm := &deleteRoundManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    logger,
				helper:    mockHelper,
				config:    mockConfig,
				operationWrapper: func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (DeleteRoundOperationResult, error)) (DeleteRoundOperationResult, error) {
					return operationFunc(ctx)
				},
				metrics: metrics,
			}

			result, err := drm.HandleDeleteRoundButton(tt.args.ctx, tt.args.i)

			if tt.expectErr {
				if err == nil && result.Error == nil {
					t.Errorf("%s: Expected error, got nil (err and result.Error)", tt.name)
				}
				if tt.expectedError != "" {
					var actualError string
					if err != nil {
						actualError = err.Error()
					} else if result.Error != nil {
						actualError = result.Error.Error()
					}
					if !strings.Contains(actualError, tt.expectedError) {
						t.Errorf("%s: Expected error containing: %q, got: %q", tt.name, tt.expectedError, actualError)
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.name, err)
				}
				if result.Error != nil {
					t.Errorf("%s: Unexpected result.Error: %v", tt.name, result.Error)
				}
			}
			if tt.expectSuccess != (result.Success != nil) {
				t.Errorf("%s: Expected success: %v, got %v", tt.name, tt.expectSuccess, result.Success != nil)
			}
		})
	}
}
