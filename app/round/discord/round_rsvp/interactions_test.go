package roundrsvp

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	helpersmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_roundRsvpManager_HandleRoundResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testRoundID := sharedtypes.RoundID(uuid.New())
	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}

	createInteraction := func(customID string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-123",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-123",
						Username: "TestUser ",
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

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
		expectedError string
	}{
		{
			name: "interaction respond error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|" + testRoundID.String()),
			},
			expectedError: "failed to respond to interaction",
		},
		{
			name: "unknown response type",
			setup: func() {
				// No mocks needed - function should return early
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("unknown_response|789"),
			},
			expectedError: "unknown response type: unknown_response|789",
		},
		{
			name: "invalid event ID",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|invalid"),
			},
			expectedError: "failed to parse round UUID",
		},
		{
			name: "create result message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(discordroundevents.RoundParticipantJoinReqTopic)).
					Return(nil, errors.New("failed to create result message")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|" + testRoundID.String()),
			},
			expectedError: "failed to create result message",
		},
		{
			name: "publish error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(discordroundevents.RoundParticipantJoinReqTopic)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundParticipantJoinReqTopic), gomock.Any()).
					Return(errors.New("failed to publish message")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|" + testRoundID.String()),
			},
			expectedError: "failed to publish message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rrm := &roundRsvpManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    mockLogger,
				helper:    mockHelper,
				config:    mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (RoundRsvpOperationResult, error)) (RoundRsvpOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			result, err := rrm.HandleRoundResponse(tt.args.ctx, tt.args.i)

			if tt.expectedError != "" {
				if err == nil && result.Error == nil {
					t.Errorf("expected error containing %q, got none (err: nil, result.Error: nil)", tt.expectedError)
				}
				if result.Error != nil && !strings.Contains(result.Error.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %v", tt.expectedError, result.Error)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_roundRsvpManager_InteractionJoinRoundLate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}

	createInteraction := func(customID string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-123",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-123",
						Username: "TestUser ",
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

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
		expectedError string
	}{
		{
			name: "invalid custom ID format",
			setup: func() {
				// No mocks needed - function should exit early
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("invalid_format"),
			},
			expectedError: "invalid custom ID format: invalid_format",
		},
		{
			name: "interaction respond error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_join|round-" + testRoundID.String()),
			},
			expectedError: "failed to respond",
		},
		{
			name: "publish error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(discordroundevents.RoundParticipantJoinReqTopic)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundParticipantJoinReqTopic), gomock.Any()).
					Return(errors.New("failed to publish")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_join|round-" + testRoundID.String()),
			},
			expectedError: "failed to publish",
		},
		{
			name: "invalid event ID",
			setup: func() {
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_join|invalid"),
			},
			expectedError: "failed to parse round UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rrm := &roundRsvpManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    mockLogger,
				helper:    mockHelper,
				config:    mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (RoundRsvpOperationResult, error)) (RoundRsvpOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			result, err := rrm.InteractionJoinRoundLate(tt.args.ctx, tt.args.i)

			if tt.expectedError != "" {
				if err == nil && result.Error == nil {
					t.Errorf("expected error containing %q, got none (err: nil, result.Error: nil)", tt.expectedError)
				}
				if result.Error != nil && !strings.Contains(result.Error.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %v", tt.expectedError, result.Error)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
