package scoreround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	helpersmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_scoreRoundManager_HandleScoreButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := uuid.New()
	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockMetrics := &discordmetrics.NoOpMetrics{}

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

	tests := []struct {
		name          string
		setup         func()
		interaction   *discordgo.InteractionCreate
		expectedError string
	}{
		{
			name: "invalid custom ID format",
			setup: func() {
				// No mocks needed - function should exit early
			},
			interaction:   createInteraction("score_button"),
			expectedError: "invalid custom ID for score button",
		},
		{
			name: "finalized round rejection",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...any) error {
						if !strings.Contains(resp.Data.Content, "finalized") {
							return errors.New("expected finalized message content")
						}
						return nil
					}).
					Times(1)
			},
			interaction:   createInteraction("score_button|" + testRoundID.String() + "|finalized"),
			expectedError: "",
		},
		{
			name: "finalized round response error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("failed to respond to interaction")).
					Times(1)
			},
			interaction:   createInteraction("score_button|" + testRoundID.String() + "|finalized"),
			expectedError: "failed to respond to interaction",
		},
		{
			name: "modal response error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...any) error {
						if resp.Type != discordgo.InteractionResponseModal {
							return errors.New("expected modal response type")
						}
						return errors.New("failed to open modal")
					}).
					Times(1)
			},
			interaction:   createInteraction("score_button|" + testRoundID.String()),
			expectedError: "failed to open modal",
		},
		{
			name: "successful modal open",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...any) error {
						if resp.Type != discordgo.InteractionResponseModal {
							return errors.New("expected modal response type")
						}
						if !strings.Contains(resp.Data.CustomID, "submit_score_modal") {
							return errors.New("expected submit_score_modal in custom ID")
						}
						return nil
					}).
					Times(1)
			},
			interaction:   createInteraction("score_button|" + testRoundID.String()),
			expectedError: "",
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
				metrics:   mockMetrics,
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			result, err := srm.HandleScoreButton(context.Background(), tt.interaction)

			if tt.expectedError != "" {
				if err == nil && (result.Error == nil || !strings.Contains(result.Error.Error(), tt.expectedError)) {
					t.Errorf("expected error containing %q, got %v", tt.expectedError, result.Error)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_scoreRoundManager_HandleScoreSubmission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := uuid.New()
	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockHelper := helpersmocks.NewMockHelpers(ctrl)
	mockMetrics := &discordmetrics.NoOpMetrics{}

	// Helper function to create a modal interaction with various configurations
	createModalInteraction := func(customID string, components []discordgo.MessageComponent) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID: "interaction-123",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID:       "user-123",
						Username: "TestUser",
					},
				},
				Data: discordgo.ModalSubmitInteractionData{
					CustomID:   customID,
					Components: components,
				},
				Type: discordgo.InteractionModalSubmit,
			},
		}
	}

	createTextInput := func(value string) *discordgo.TextInput {
		return &discordgo.TextInput{
			CustomID: "score_input",
			Value:    value,
		}
	}

	tests := []struct {
		name          string
		setup         func()
		interaction   *discordgo.InteractionCreate
		expectedError string
	}{
		{
			name: "invalid modal custom ID",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			interaction: createModalInteraction("invalid_modal_id", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
			expectedError: "invalid modal custom ID",
		},
		{
			name: "invalid round ID",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|invalid-uuid|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
			expectedError: "invalid round ID",
		},
		{
			name: "interaction acknowledge error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to acknowledge interaction")).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
			expectedError: "failed to acknowledge interaction",
		},
		{
			name: "missing score input",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, _ bool, params *discordgo.WebhookParams, _ ...any) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Could not read your score") {
							return nil, errors.New("expected error message about score input")
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			interaction:   createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{}),
			expectedError: "could not extract score input",
		},
		{
			name: "invalid score format",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, _ bool, params *discordgo.WebhookParams, _ ...any) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Invalid score") {
							return nil, errors.New("expected error message about invalid score")
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("not-a-number"),
					},
				},
			}),
			expectedError: "invalid score input",
		},
		{
			name: "create result message error",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundScoreUpdateRequest)).
					Return(nil, errors.New("failed to create result message")).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, _ bool, params *discordgo.WebhookParams, _ ...any) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Something went wrong") {
							return nil, errors.New("expected generic error message")
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
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
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundScoreUpdateRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScoreUpdateRequest), gomock.Any()).
					Return(errors.New("failed to publish message")).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, _ bool, params *discordgo.WebhookParams, _ ...any) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Failed to submit") {
							return nil, errors.New("expected failure message")
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
			expectedError: "failed to publish message",
		},
		{
			name: "successful score submission",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundScoreUpdateRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScoreUpdateRequest), gomock.Any()).
					Return(nil).
					Times(1)

				// For trace event
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundTraceEvent)).
					Return(&message.Message{UUID: "trace-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundTraceEvent), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *discordgo.Interaction, _ bool, params *discordgo.WebhookParams, _ ...any) (*discordgo.Message, error) {
						if !strings.Contains(params.Content, "Your score of 5 has been submitted") {
							return nil, errors.New("expected success message with score")
						}
						return &discordgo.Message{}, nil
					}).
					Times(1)
			},
			interaction: createModalInteraction("submit_score_modal|"+testRoundID.String()+"|user-123", []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						createTextInput("5"),
					},
				},
			}),
			expectedError: "",
		},
		{
			name: "successful submission with alternate ActionsRow type",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundScoreUpdateRequest)).
					Return(&message.Message{UUID: "msg-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScoreUpdateRequest), gomock.Any()).
					Return(nil).
					Times(1)

				// For trace event
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Eq(roundevents.RoundTraceEvent)).
					Return(&message.Message{UUID: "trace-123"}, nil).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Eq(roundevents.RoundTraceEvent), gomock.Any()).
					Return(nil).
					Times(1)

				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					Times(1)
			},
			// Using the same interaction but a different component type case from the code
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-123",
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID:       "user-123",
							Username: "TestUser",
						},
					},
					Data: discordgo.ModalSubmitInteractionData{
						CustomID: "submit_score_modal|" + testRoundID.String() + "|user-123",
						Components: []discordgo.MessageComponent{
							&discordgo.ActionsRow{
								Components: []discordgo.MessageComponent{
									&discordgo.TextInput{
										CustomID: "score_input",
										Value:    "-3",
									},
								},
							},
						},
					},
					Type: discordgo.InteractionModalSubmit,
				},
			},
			expectedError: "",
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
				metrics:   mockMetrics,
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			result, _ := srm.HandleScoreSubmission(context.Background(), tt.interaction)

			if tt.expectedError != "" {
				// Check only the Error field in the result
				if result.Error == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(result.Error.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, result.Error.Error())
				}
			} else if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}
		})
	}
}
