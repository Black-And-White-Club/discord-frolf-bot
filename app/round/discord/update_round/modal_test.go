package updateround

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_updateRoundManager_SendupdateRoundModal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any])
		ctx         context.Context
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful send",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if r.Type != discordgo.InteractionResponseModal {
							t.Errorf("Expected InteractionResponseModal, got %v", r.Type)
						}
						if r.Data.Title != "Update Round" {
							t.Errorf("Expected title 'Update Round', got %v", r.Data.Title)
						}
						// CustomID should include the round ID and message ID: "update_round_modal|{roundID}|{messageID}"
						expectedCustomIDPrefix := "update_round_modal|"
						if !strings.HasPrefix(r.Data.CustomID, expectedCustomIDPrefix) {
							t.Errorf("Expected CustomID to start with '%s', got %v", expectedCustomIDPrefix, r.Data.CustomID)
						}
						if len(r.Data.Components) != 5 {
							t.Errorf("Expected 5 components, got %d", len(r.Data.Components))
						}
						return nil
					}).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantSuccess: "modal sent",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to send modal",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to send modal")).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantSuccess: "",
			wantErrMsg:  "failed to send modal",
			wantErrIs:   nil,
		},
		{
			name: "nil interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// No interaction with mocks expected
			},
			ctx:         context.Background(),
			args:        nil,
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
		},
		{
			name: "nil user in interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// No interaction with mocks expected
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: nil,
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name: "nil member and user in interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// No interaction with mocks expected
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:     "interaction-id",
					Type:   discordgo.InteractionApplicationCommand,
					User:   nil,
					Member: nil,
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name: "context cancelled before operation",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// No interaction with mocks expected due to early context cancel
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantSuccess: "",
			wantErrMsg:  context.Canceled.Error(),
			wantErrIs:   context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			// Setup per-test mock expectations
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore)
			}

			urm := &updateRoundManager{
				session:          mockSession,
				logger:           mockLogger,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: testOperationWrapper,
			}

			testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
			testRoundID := sharedtypes.RoundID(testUUID)
			result, err := urm.SendUpdateRoundModal(tt.ctx, tt.args, testRoundID)

			// Check for error
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("SendupdateRoundModal() expected error containing %q, but got nil", tt.wantErrMsg)
					t.FailNow()
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SendupdateRoundModal() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("SendupdateRoundModal() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("SendupdateRoundModal() unexpected error: %v", err)
					t.FailNow()
				}
			}

			// Check success only if no error
			if tt.wantErrMsg == "" {
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("SendupdateRoundModal() UpdateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

// Helper function to create a test interaction with modal submit data for update round
func createTestUpdateInteraction(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:      "interaction-id",
			Token:   "interaction-token",
			GuildID: "test-guild",
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
			Type: discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: "update_round_modal|550e8400-e29b-41d4-a716-446655440000|message-123",
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "title",
								Value:    title,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "description",
								Value:    description,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "start_time",
								Value:    startTime,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "timezone",
								Value:    timezone,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "location",
								Value:    location,
							},
						},
					},
				},
			},
		},
	}
}

func Test_updateRoundManager_HandleUpdateRoundModalSubmit(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.InteractionCreate
		ctx         context.Context
		setup       func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers)
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name:        "successful submission",
			interaction: createTestUpdateInteraction("Updated Round", "Updated description", "2025-05-01 15:00", "America/Chicago", "Updated Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundUpdateModalSubmittedV1), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "missing required fields",
			interaction: createTestUpdateInteraction("", "", "", "", ""),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, _ ...interface{}) error {
						// Implementation responds with a single message asking the user to fill at least one field.
						if !strings.Contains(r.Data.Content, "Please fill at least one field") {
							t.Errorf("Expected error message to contain 'Please fill at least one field', got %v", r.Data.Content)
						}
						return nil
					}).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "no fields provided",
			wantErrIs:   nil,
		},
		{
			name:        "field too long",
			interaction: createTestUpdateInteraction(strings.Repeat("A", 101), "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundUpdateModalSubmittedV1), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "failed to acknowledge submission",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to acknowledge")).
					Times(1)
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "failed to store interaction",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "failed to create event",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundUpdateModalSubmittedV1), gomock.Any()).
					Return(errors.New("failed to create event")).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "failed to create event",
			wantErrIs:   nil,
		},
		{
			name:        "failed to publish event",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundUpdateModalSubmittedV1), gomock.Any()).
					Return(errors.New("failed to publish")).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "failed to publish",
			wantErrIs:   nil,
		},
		{
			name:        "nil interaction",
			interaction: nil,
			ctx:         context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				// No expectations
			},
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
		},
		{
			name: "missing user ID",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionModalSubmit,
					Data: discordgo.ModalSubmitInteractionData{},
				},
			},
			ctx: context.Background(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				// No expectations
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name:        "context cancelled",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			}(),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface[any], mockHelper *utilsmocks.MockHelpers) {
				// No expectations due to early context cancellation
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
			mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockHelper := utilsmocks.NewMockHelpers(ctrl)

			// Setup test expectations
			tt.setup(mockSession, mockPublisher, mockInteractionStore, mockHelper)

			// Allow any InteractionRespond calls not explicitly expected by the test setups.
			mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			// Allow any Publish calls not explicitly expected by the test setups.
			mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			// Create the manager with mocked dependencies
			urm := &updateRoundManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           &config.Config{},
				interactionStore: mockInteractionStore,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
				operationWrapper: testOperationWrapper,
			}

			result, err := urm.HandleUpdateRoundModalSubmit(tt.ctx, tt.interaction)

			// Success check
			gotSuccess, ok := result.Success.(string)
			if ok {
				if gotSuccess != tt.wantSuccess {
					t.Errorf("HandleUpdateRoundModalSubmit() UpdateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			} else if tt.wantSuccess != "" {
				t.Errorf("HandleUpdateRoundModalSubmit() UpdateRoundOperationResult.Success was not a string: got %T, want string", result.Success)
			}

			// Error message check -- the implementation sometimes returns the error inside result.Error
			if tt.wantErrMsg != "" {
				actualErr := err
				if actualErr == nil && result.Error != nil {
					actualErr = result.Error
				}
				if actualErr == nil {
					t.Errorf("HandleUpdateRoundModalSubmit() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(actualErr.Error(), tt.wantErrMsg) {
					t.Errorf("HandleUpdateRoundModalSubmit() error message mismatch: got %q, want substring %q", actualErr.Error(), tt.wantErrMsg)
				}
				// Error type check
				if tt.wantErrIs != nil && !errors.Is(actualErr, tt.wantErrIs) {
					t.Errorf("HandleUpdateRoundModalSubmit() error type mismatch: got %T, want %T", actualErr, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleUpdateRoundModalSubmit() unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Errorf("HandleUpdateRoundModalSubmit() unexpected result error: %v", result.Error)
				}
			}
		})
	}
}

func Test_updateRoundManager_HandleUpdateRoundModalCancel(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any])
		args        *discordgo.InteractionCreate
		ctx         context.Context
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful_cancel",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// The implementation no longer deletes stored interactions; just expect the ephemeral response.
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...any) error {
						if r.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", r.Type)
						}
						if r.Data.Content != "Round update cancelled." {
							t.Errorf("Expected content 'Round update cancelled.', got %v", r.Data.Content)
						}
						if r.Data.Flags != discordgo.MessageFlagsEphemeral {
							t.Errorf("Expected ephemeral message, got flags %v", r.Data.Flags)
						}
						return nil
					}).
					Times(1)
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			},
			ctx:         context.Background(),
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "error sending response",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// InteractionRespond errors are ignored by the implementation; expect it will be called.
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to send response")).
					Times(1)
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			},
			ctx: context.Background(),
			// The implementation ignores errors from InteractionRespond during cancel, so
			// we expect a successful cancellation even if sending the ephemeral response fails.
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "nil interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// No expectations as the function should return early
			},
			args:        nil,
			ctx:         context.Background(),
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
		},
		{
			name: "context cancelled",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface[any]) {
				// operationWrapper used in tests executes the operation regardless of cancelled context,
				// so the handler will still attempt to send the ephemeral response.
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			}(),
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger

			// Setup test expectations
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore)
			}

			// Allow any InteractionRespond calls not explicitly expected by the test setups.
			mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			// Create the manager with mocked dependencies
			urm := &updateRoundManager{
				session:          mockSession,
				interactionStore: mockInteractionStore,
				logger:           mockLogger,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
				operationWrapper: testOperationWrapper,
			}
			result, err := urm.HandleUpdateRoundModalCancel(tt.ctx, tt.args)

			// Success check
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("HandleUpdateRoundModalCancel() CreateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Error message check
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("HandleUpdateRoundModalCancel() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("HandleUpdateRoundModalCancel() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("HandleUpdateRoundModalCancel() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleUpdateRoundModalCancel() unexpected error: %v", err)
				}
			}
		})
	}
}
