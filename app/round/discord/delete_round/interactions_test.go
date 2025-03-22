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
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_deleteRoundManager_HandleDeleteRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}

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
	createExpectedPayload := func(roundID int64) roundevents.RoundDeleteRequestPayload {
		return roundevents.RoundDeleteRequestPayload{
			RoundID:              roundtypes.ID(roundID),
			RequestingUserUserID: "user-456",
		}
	}

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
	}{
		{
			name: "successful delete request",
			setup: func() {
				// Mock InteractionRespond call
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseDeferredMessageUpdate {
							t.Errorf("Expected InteractionResponseDeferredMessageUpdate, got %v", ir.Type)
						}
						return nil
					}).
					Times(1)

				expectedPayload := createExpectedPayload(789)
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
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the message is ephemeral
						if params.Flags != discordgo.MessageFlagsEphemeral {
							t.Errorf("Expected MessageFlagsEphemeral, got %v", params.Flags)
						}
						if !strings.Contains(params.Content, "deletion request sent") {
							t.Errorf("Expected message to contain 'deletion request sent', got %s", params.Content)
						}
						return &discordgo.Message{ID: "message-456"}, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|789"),
			},
		},
		{
			name: "invalid custom ID format",
			setup: func() {
				// No mocks needed - function should return early
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete"), // Missing the round ID part
			},
		},
		{
			name: "invalid round ID",
			setup: func() {
				// No mocks needed - function should return early
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|invalid"),
			},
		},
		{
			name: "interaction respond error",
			setup: func() {
				// Mock InteractionRespond call with error
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|789"),
			},
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
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Failed to delete") {
							t.Errorf("Expected error message about failing to delete, got %s", params.Content)
						}
						return &discordgo.Message{ID: "message-456"}, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|789"),
			},
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
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Failed to delete") {
							t.Errorf("Expected error message about failing to delete, got %s", params.Content)
						}
						return &discordgo.Message{ID: "message-456"}, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|789"),
			},
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
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|789"),
			},
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
				logger:    mockLogger,
				helper:    mockHelper,
				config:    mockConfig,
			}

			drm.HandleDeleteRound(tt.args.ctx, tt.args.i)
		})
	}
}
