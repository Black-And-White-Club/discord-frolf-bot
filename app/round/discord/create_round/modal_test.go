package createround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_createRoundManager_SendCreateRoundModal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface)
		ctx         context.Context
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful send",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if r.Type != discordgo.InteractionResponseModal {
							t.Errorf("Expected InteractionResponseModal, got %v", r.Type)
						}
						if r.Data.Title != "Create Round" {
							t.Errorf("Expected title 'Create Round', got %v", r.Data.Title)
						}
						if r.Data.CustomID != "create_round_modal" {
							t.Errorf("Expected CustomID 'create_round_modal', got %v", r.Data.CustomID)
						}
						if len(r.Data.Components) != 5 {
							t.Errorf("Expected 5 components, got %v", r.Type)
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
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
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
			wantErrMsg:  "failed to send create round modal: failed to send modal",
			wantErrIs:   nil,
		},
		{
			name: "nil interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
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
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
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
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
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
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
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
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			// Setup per-test mock expectations
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore)
			}

			crm := &createRoundManager{
				session:          mockSession,
				logger:           mockLogger,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: testOperationWrapper,
			}

			result, err := crm.SendCreateRoundModal(tt.ctx, tt.args)

			// Check for error
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("SendCreateRoundModal() expected error containing %q, but got nil", tt.wantErrMsg)
					t.FailNow()
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SendCreateRoundModal() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("SendCreateRoundModal() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("SendCreateRoundModal() unexpected error: %v", err)
					t.FailNow()
				}
			}

			// Check success only if no error
			if tt.wantErrMsg == "" {
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("SendCreateRoundModal() CreateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

func Test_createRoundManager_HandleCreateRoundModalSubmit(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.InteractionCreate
		ctx         context.Context
		setup       func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface)
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name:        "successful submission",
			interaction: createTestInteraction("Test Round", "Fun round description", "2025-04-01 14:00", "America/Chicago", "Disc Golf Park"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(sharedroundevents.RoundCreateModalSubmittedV1), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "round creation request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "missing required fields",
			interaction: createTestInteraction("", "", "", "", ""),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "validation failed: Title is required. Start Time is required.",
			wantErrIs:   nil,
		},
		{
			name:        "default timezone",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(sharedroundevents.RoundCreateModalSubmittedV1), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "round creation request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "field too long",
			interaction: createTestInteraction(strings.Repeat("A", 101), "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "validation failed: Title must be less than 100 characters.",
			wantErrIs:   nil,
		},
		{
			name:        "failed to acknowledge submission",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to acknowledge")).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "failed to acknowledge submission",
			wantErrIs:   nil,
		},
		{
			name:        "failed to store interaction",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("failed to store")).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "failed to store interaction",
			wantErrIs:   nil,
		},
		{
			name:        "failed to publish event",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockPublisher.EXPECT().
					Publish(gomock.Eq(sharedroundevents.RoundCreateModalSubmittedV1), gomock.Any()).
					Return(errors.New("failed to publish")).
					Times(1)
			},
			wantSuccess: "",
			wantErrMsg:  "failed to publish event",
			wantErrIs:   nil,
		},
		{
			name:        "nil interaction",
			interaction: nil,
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
			},
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
		},
		{
			name:        "missing user info",
			interaction: &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionModalSubmit, Data: discordgo.ModalSubmitInteractionData{CustomID: "create_round_modal"}}},
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name:        "context cancelled before operation",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
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
			mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockHelper := utilsmocks.NewMockHelpers(ctrl)

			// Setup test expectations
			tt.setup(mockSession, mockPublisher, mockInteractionStore)

			// Create config with necessary values
			testConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "test-guild",
				},
			}

			// Create the manager with mocked dependencies
			crm := &createRoundManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           testConfig,
				interactionStore: mockInteractionStore,
				operationWrapper: testOperationWrapper,
			}

			ctx := context.Background()
			if ctxOverride := tt.ctx; ctxOverride != nil {
				ctx = ctxOverride
			}

			result, err := crm.HandleCreateRoundModalSubmit(ctx, tt.interaction)

			// Success check
			gotSuccess, ok := result.Success.(string)
			if ok {
				if gotSuccess != tt.wantSuccess {
					t.Errorf("HandleCreateRoundModalSubmit() CreateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			} else if tt.wantSuccess != "" {
				t.Errorf("HandleCreateRoundModalSubmit() CreateRoundOperationResult.Success was not a string: got %T, want string", result.Success)
			}

			// Error message check
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("HandleCreateRoundModalSubmit() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("HandleCreateRoundModalSubmit() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("HandleCreateRoundModalSubmit() unexpected error: %v", err)
				}
			}

			// Error type check
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("HandleCreateRoundModalSubmit() error type mismatch: got %T, want %T", err, tt.wantErrIs)
			}

			// Result error check
			if tt.wantErrMsg == "" && result.Error != nil {
				t.Errorf("HandleCreateRoundModalSubmit() CreateRoundOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
			}

			// Check for context cancellation error in result
			if tt.name == "context cancelled before operation" {
				if !errors.Is(result.Error, context.Canceled) {
					t.Errorf("HandleCreateRoundModalSubmit() CreateRoundOperationResult.Error mismatch for cancelled context: got %v, want %v", result.Error, context.Canceled)
				}
			}
		})
	}
}

// Helper function to create a test interaction with modal submit data
func createTestInteraction(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
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
				CustomID: "create_round_modal",
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

func Test_createRoundManager_HandleCreateRoundModalCancel(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface)
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
		ctx         context.Context // Added context to test cases
	}{
		{
			name: "successful_cancel",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Delete("interaction-id").
					Times(1)

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
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
			wantSuccess: "round creation cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
			ctx:         context.Background(),
		},
		{
			name: "error deleting interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Delete("interaction-id").
					Times(1)

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
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
			wantSuccess: "round creation cancelled", //  Expect success, not an error
			wantErrMsg:  "",                         //  No error expected
			wantErrIs:   nil,
			ctx:         context.Background(),
		},
		{
			name: "error sending response",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				// Expect interaction store to be called
				mockInteractionStore.EXPECT().
					Delete("interaction-id").
					Times(1)

				// Expect error when sending response
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
			wantSuccess: "",
			wantErrMsg:  "failed to send cancellation response",
			wantErrIs:   nil,
			ctx:         context.Background(),
		},
		{
			name: "nil interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				// No expectations as the function should return early
			},
			args:        nil,
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
			ctx:         context.Background(),
		},
		{
			name: "context cancelled",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				//  The function should not call delete or respond in this case.
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "cancelled-id",
				},
			},
			wantSuccess: "",
			wantErrMsg:  context.Canceled.Error(),
			wantErrIs:   nil,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)

			// Setup test expectations
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore)
			}

			// Create the manager with mocked dependencies
			crm := &createRoundManager{
				session:          mockSession,
				interactionStore: mockInteractionStore,
				logger:           loggerfrolfbot.NoOpLogger,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
				operationWrapper: testOperationWrapper,
			}

			result, err := crm.HandleCreateRoundModalCancel(tt.ctx, tt.args)

			// Success check
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("HandleCreateRoundModalCancel() CreateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Error message check
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("HandleCreateRoundModalCancel() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("HandleCreateRoundModalCancel() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("HandleCreateRoundModalCancel() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleCreateRoundModalCancel() unexpected error: %v", err)
				}
			}
		})
	}
}
