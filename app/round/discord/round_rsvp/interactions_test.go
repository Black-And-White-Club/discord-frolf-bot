package roundrsvp

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
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

func Test_roundRsvpManager_HandleRoundResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockConfig := &config.Config{}

	// Helper function to create a sample InteractionCreate with the desired custom ID
	createInteraction := func(customID string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-123",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-123",
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

	// Helper function to create an expected participant join request payload
	createExpectedPayload := func(response roundtypes.Response) roundevents.ParticipantJoinRequestPayload {
		tagNumber := 0
		return roundevents.ParticipantJoinRequestPayload{
			RoundID:   789, // Based on the "round_accept|789" CustomID
			UserID:    "user-123",
			Response:  response,
			TagNumber: &tagNumber,
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
			name: "successful accept response",
			setup: func() {
				// Fix for the InteractionRespond mock setup
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseDeferredMessageUpdate {
							t.Errorf("Expected InteractionResponseDeferredMessageUpdate, got %v", ir.Type)
						}
						return nil
					}).
					Times(1)

				expectedPayload := createExpectedPayload(roundtypes.ResponseAccept)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedPayload), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundParticipantJoinRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the message is ephemeral
						if params.Flags != discordgo.MessageFlagsEphemeral {
							t.Errorf("Expected MessageFlagsEphemeral, got %v", params.Flags)
						}
						if !strings.Contains(params.Content, "You have chosen") {
							t.Errorf("Expected message to contain 'You have chosen', got %s", params.Content)
						}
						return &discordgo.Message{ID: "message-123"}, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|789"),
			},
		},
		{
			name: "successful decline response",
			setup: func() {
				// Fix for the InteractionRespond mock setup
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				expectedPayload := createExpectedPayload(roundtypes.ResponseDecline)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedPayload), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundParticipantJoinRequest), gomock.Any()).
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true),
						gomock.AssignableToTypeOf(&discordgo.WebhookParams{
							Flags: discordgo.MessageFlagsEphemeral,
						}),
						gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_decline|789"),
			},
		},
		{
			name: "successful tentative response",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				expectedPayload := createExpectedPayload(roundtypes.ResponseTentative)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedPayload), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundParticipantJoinRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true),
						gomock.AssignableToTypeOf(&discordgo.WebhookParams{
							Flags: discordgo.MessageFlagsEphemeral,
						}),
						gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_tentative|789"),
			},
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
		},
		{
			name: "interaction respond error",
			setup: func() {
				// Mock InteractionRespond call with error
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
				i:   createInteraction("round_accept|789"),
			},
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
		},
		{
			name: "create result message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(nil, errors.New("failed to create result message")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|789"),
			},
		},
		{
			name: "publish error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundParticipantJoinRequest), gomock.Any()).
					Return(errors.New("failed to publish message")).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|789"),
			},
		},
		{
			name: "followup message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundParticipantJoinRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundParticipantJoinRequest), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), gomock.Eq(true),
						gomock.AssignableToTypeOf(&discordgo.WebhookParams{
							Flags: discordgo.MessageFlagsEphemeral,
						}),
						gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_accept|789"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rrm := &roundRsvpManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			rrm.HandleRoundResponse(tt.args.ctx, tt.args.i)
		})
	}
}
