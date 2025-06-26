package signup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_signupManager_SendSignupModal(t *testing.T) {
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
					Return(nil).Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "12345"},
				},
			},
			wantSuccess: "modal sent",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to send modal",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("send error")).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "12345"},
				},
			},
			wantSuccess: "",
			wantErrMsg:  "send error",
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
			name: "nil user",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface) {
				// No interaction with mocks expected
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: nil,
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user is nil in interaction",
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
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "12345"},
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
			mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
			mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			mockConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "guild_123",
				},
			}

			// Setup per-test mock expectations
			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore)
			}

			sm := &signupManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           logger,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: testOperationWrapper,
			}

			result, err := sm.SendSignupModal(tt.ctx, tt.args)

			// Success field check
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("SignupOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Error message/content checks
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("SendSignupModal() expected error containing %q, but got nil", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SendSignupModal() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("SendSignupModal() unexpected error: %v", err)
				}
			}

			// Error type check
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("SendSignupModal() error type mismatch: got %T, want %T", err, tt.wantErrIs)
			}

			// Result error should match expected if set
			if tt.wantErrMsg == "" && result.Error != nil {
				t.Errorf("SignupOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
			}

			// Special context cancel case
			if tt.name == "context cancelled before operation" {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("SendSignupModal() error mismatch for cancelled context: got %v, want %v", err, context.Canceled)
				}
				if result.Error == nil || !errors.Is(result.Error, context.Canceled) {
					t.Errorf("SignupOperationResult.Error mismatch for cancelled context: got %v, want %v", result.Error, context.Canceled)
				}
			}
		})
	}
}

func Test_signupManager_HandleSignupModalSubmit(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus)
		ctx         context.Context
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
		shouldPanic bool
	}{
		{
			name: "successful submission",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx:         context.Background(),
			args:        validInteraction("123"),
			wantSuccess: "signup event published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to acknowledge submission",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed to acknowledge modal submission: %w", errors.New("acknowledge error"))).
					Times(1)
			},
			ctx:         context.Background(),
			args:        validInteraction("123"),
			wantSuccess: "",
			wantErrMsg:  "failed to acknowledge modal submission",
			wantErrIs:   nil,
		},
		{
			name: "invalid tag number format",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect a followup message for the error
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					Return(nil, nil).
					Times(1)
			},
			ctx:         context.Background(),
			args:        validInteraction("abc"), // Non-numeric tag
			wantSuccess: "",
			wantErrMsg:  "tag number must be a valid number, received 'abc'",
			wantErrIs:   nil,
		},
		{
			name: "failed to publish event",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed to publish signup event: %w", errors.New("publish error"))).
					Times(1)

				// Expect a followup message for the error
				mockSession.EXPECT().
					FollowupMessageCreate(gomock.Any(), true, gomock.Any()).
					Return(nil, nil).
					Times(1)
			},
			ctx:         context.Background(),
			args:        validInteraction("123"),
			wantSuccess: "",
			wantErrMsg:  "failed to publish signup event",
			wantErrIs:   nil,
		},
		{
			name: "nil_interaction",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
			},
			ctx:         context.Background(),
			args:        nil, // Passing nil as the interaction
			wantSuccess: "",
			wantErrMsg:  "interaction is nil or incomplete",
			wantErrIs:   nil,
		},
		{
			name: "missing user info",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    uuid.New().String(),
					Token: uuid.New().String(),
					Type:  discordgo.InteractionModalSubmit,
					Data: discordgo.ModalSubmitInteractionData{
						CustomID: "signup_modal",
					},
					Member: &discordgo.Member{
						User: nil, // Missing user info
					},
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name: "context cancelled before operation",
			setup: func(mockSession *discordmocks.MockSession, mockInteractionStore *storagemocks.MockISInterface, mockPublisher *eventbusmocks.MockEventBus) {
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately
				return ctx
			}(),
			args:        validInteraction("123"),
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
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			mockConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "guild_123",
				},
			}

			if tt.setup != nil {
				tt.setup(mockSession, mockInteractionStore, mockPublisher)
			}

			sm := &signupManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           logger,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: testOperationWrapper,
			}

			// Use defer/recover to handle potential panics
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("Test %s panicked unexpectedly: %v", tt.name, r)
					}
				} else if tt.shouldPanic {
					t.Errorf("Expected panic, but test %s did not panic", tt.name)
				}
			}()

			result, err := sm.HandleSignupModalSubmit(tt.ctx, tt.args)

			// Check the direct error return
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("HandleSignupModalSubmit() error was nil, expected error containing %q", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("HandleSignupModalSubmit() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("HandleSignupModalSubmit() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleSignupModalSubmit() error was not nil, expected nil. Got: %v", err)
				}
			}

			// Check the SignupOperationResult.Success field
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("SignupOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Check the SignupOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Logf("SignupOperationResult.Error is nil, which might be expected if the error is only in the direct return")
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("SignupOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
					t.Errorf("SignupOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
				}
			} else {
				if result.Error != nil {
					t.Errorf("SignupOperationResult.Error was not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}

func validInteraction(tagValue string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    uuid.New().String(),
			Token: uuid.New().String(),
			Type:  discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: "signup_modal",
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "tag_number",
								Value:    tagValue,
							},
						},
					},
				},
			},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "12345"},
			},
		},
	}
}
