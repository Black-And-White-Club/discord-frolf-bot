package scoreround

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	helpersmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_scoreRoundManager_HandleScoreButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}

	// Helper function to create a sample interaction with the desired custom ID
	createButtonInteraction := func(roundID string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "button-interaction-123",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-123",
						Username: "TestUser",
					},
				},
				Data: discordgo.MessageComponentInteractionData{
					CustomID:      fmt.Sprintf("score_button|%s", roundID),
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
	}{
		{
			name: "successful modal opening",
			setup: func() {
				// Expected logger calls
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Opening score submission modal"), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseModal {
							t.Errorf("Expected InteractionResponseModal, got %v", ir.Type)
						}

						if ir.Data.Title != "Submit Your Score" {
							t.Errorf("Expected 'Submit Your Score', got %s", ir.Data.Title)
						}

						expectedCustomID := fmt.Sprintf("submit_score_modal|%s|%s", "789", "user-123")
						if ir.Data.CustomID != expectedCustomID {
							t.Errorf("Expected CustomID %s, got %s", expectedCustomID, ir.Data.CustomID)
						}

						return nil
					}).
					Times(1)

				// Expected success logger call
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Successfully opened score modal"), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createButtonInteraction("789"),
			},
		},
		{
			name: "interaction respond error",
			setup: func() {
				// Expected logger calls
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Opening score submission modal"), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)

				// Expected error logger call
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Failed to open score modal"), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createButtonInteraction("789"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			srm := &scoreRoundManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    mockLogger,
				helper:    mockHelper,
				config:    mockConfig,
			}

			srm.HandleScoreButton(tt.args.ctx, tt.args.i)
		})
	}
}

func Test_scoreRoundManager_HandleScoreSubmission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}

	// Create a sample modal interaction with customizable values
	createModalInteraction := func(customID string, scoreValue string) *discordgo.InteractionCreate {
		textInput := &discordgo.TextInput{
			CustomID: "score_input",
			Value:    scoreValue,
		}

		actionsRow := &discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{textInput},
		}

		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-123",
				Message: &discordgo.Message{
					ID: "message-123",
				},
				ChannelID: "channel-123",
				Type:      discordgo.InteractionModalSubmit,
				Data: discordgo.ModalSubmitInteractionData{
					CustomID:   customID,
					Components: []discordgo.MessageComponent{actionsRow},
				},
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
	}{
		{
			name: "successful score submission",
			setup: func() {
				// Interaction acknowledge response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseDeferredChannelMessageWithSource, got %v", ir.Type)
						}
						return nil
					}).
					Times(1)

				// Logger for processing score submission
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Processing score submission"),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Create a mock message object instead of a string
				mockResultMsg := &message.Message{}

				// Create result message for score update
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq("round.score.update.request")).
					DoAndReturn(func(msg interface{}, payload interface{}, topic string) (*message.Message, error) {
						// Validate payload
						p, ok := payload.(roundevents.ScoreUpdateRequestPayload)
						if !ok {
							t.Error("Expected ScoreUpdateRequestPayload type")
						}
						if p.RoundID != 789 {
							t.Errorf("Expected RoundID 789, got %v", p.RoundID)
						}
						if p.Participant != "user-123" {
							t.Errorf("Expected Participant user-123, got %v", p.Participant)
						}
						if *p.Score != 5 {
							t.Errorf("Expected Score 5, got %v", *p.Score)
						}
						return mockResultMsg, nil
					}).
					Times(1)

				// Publish the score update
				mockPublisher.EXPECT().
					Publish(gomock.Eq("round.score.update.request"), mockResultMsg).
					Return(nil).
					Times(1)

				// Create trace event message
				mockTraceMsg := &message.Message{}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq("round.trace.event")).
					Return(mockTraceMsg, nil).
					Times(1)

				// Publish trace event
				mockPublisher.EXPECT().
					Publish(gomock.Eq("round.trace.event"), mockTraceMsg).
					Times(1)

				// Send confirmation to user
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if params.Content != "Your score of 5 has been submitted! You'll receive a confirmation once it's processed." {
							t.Errorf("Unexpected confirmation message: %s", params.Content)
						}
						return nil, nil
					}).
					Times(1)

				// Final success log
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Score submission processed successfully"),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", "5"),
			},
		},
		{
			name: "invalid modal custom ID",
			setup: func() {
				// Expected logger for invalid custom ID
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Invalid modal custom ID"), gomock.Any(), gomock.Any()).
					Times(1)

				// Expected error response to user
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", ir.Type)
						}
						if ir.Data.Content != "Something went wrong with your submission. Please try again." {
							t.Errorf("Unexpected error message: %s", ir.Data.Content)
						}
						return nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("invalid_custom_id", "5"),
			},
		},
		{
			name: "invalid round ID",
			setup: func() {
				// Expected logger for invalid round ID
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Invalid round ID"), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Expected error response to user
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, ir *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
						if ir.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", ir.Type)
						}
						if ir.Data.Content != "Invalid round information. Please try again." {
							t.Errorf("Unexpected error message: %s", ir.Data.Content)
						}
						return nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|not_a_number|user-123", "5"),
			},
		},
		{
			name: "error acknowledging interaction",
			setup: func() {
				// Interaction acknowledge error
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)

				// Expected error logger
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Failed to acknowledge score submission"), gomock.Any(), gomock.Any()).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", "5"),
			},
		},
		{
			name: "empty score input",
			setup: func() {
				// Interaction acknowledge response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expected error logger
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Could not extract score input"), gomock.Any()).
					Times(1)

				// Error response to user
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if params.Content != "Could not read your score. Please try again." {
							t.Errorf("Unexpected error message: %s", params.Content)
						}
						return nil, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", ""),
			},
		},
		{
			name: "invalid score format",
			setup: func() {
				// Interaction acknowledge response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expected error logger
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Invalid score input"), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Error response to user
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if params.Content != "Invalid score. Please enter a valid number (e.g., -3, 0, +5)." {
							t.Errorf("Unexpected error message: %s", params.Content)
						}
						return nil, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", "not_a_number"),
			},
		},
		{
			name: "error creating result message",
			setup: func() {
				// Interaction acknowledge response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Logger for processing score submission
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Processing score submission"),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Create result message error
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq("round.score.update.request")).
					Return(nil, errors.New("failed to create message")).
					Times(1)

				// Expected error logger
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Failed to create result message"), gomock.Any(), gomock.Any()).
					Times(1)

				// Error response to user
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if params.Content != "Something went wrong while submitting your score. Please try again later." {
							t.Errorf("Unexpected error message: %s", params.Content)
						}
						return nil, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", "5"),
			},
		},
		{
			name: "error publishing message",
			setup: func() {
				// Interaction acknowledge response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Logger for processing score submission
				mockLogger.EXPECT().
					Info(gomock.Any(), gomock.Eq("Processing score submission"),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Create a mock message object
				mockResultMsg := &message.Message{}

				// Create result message success
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq("round.score.update.request")).
					Return(mockResultMsg, nil).
					Times(1)

				// Publish error
				mockPublisher.EXPECT().
					Publish(gomock.Eq("round.score.update.request"), mockResultMsg).
					Return(errors.New("failed to publish message")).
					Times(1)

				// Expected error logger
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Eq("Failed to publish score update request"), gomock.Any(), gomock.Any()).
					Times(1)

				// Error response to user
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
						if params.Content != "Failed to submit your score. Please try again later." {
							t.Errorf("Unexpected error message: %s", params.Content)
						}
						return nil, nil
					}).
					Times(1)
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createModalInteraction("submit_score_modal|789|user-123", "5"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			srm := &scoreRoundManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    mockLogger,
				helper:    mockHelper,
				config:    mockConfig,
			}

			srm.HandleScoreSubmission(tt.args.ctx, tt.args.i)
		})
	}
}
