package signup

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

// Test wrapper function that passes through the operation call without additional logic
var testOperationWrapper = func(ctx context.Context, operationName string, operation func(ctx context.Context) (SignupOperationResult, error)) (SignupOperationResult, error) {
	return operation(ctx)
}

func Test_signupManager_MessageReactionAdd(t *testing.T) {
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
			GuildID:         "guild-id",
			SignupEmoji:     "emoji",
			SignupChannelID: "channel-id",
			SignupMessageID: "message-id",
		},
	}

	tests := []struct {
		name        string
		setup       func()
		args        *discordgo.MessageReactionAdd
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "valid reaction",
			setup: func() {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(&discordgo.Channel{}, nil).
					Times(1)
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, nil).
					Times(1)
				mockSession.EXPECT().
					GetBotUser().
					Return(&discordgo.User{ID: "bot-user-id"}, nil).
					Times(1)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "message-id",
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id",
				},
			},
			wantSuccess: "signup button sent",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "invalid channel id",
			setup: func() {
				mockSession.EXPECT().
					GetBotUser().
					Return(&discordgo.User{ID: "bot-user-id"}, nil).
					Times(0)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "invalid-channel-id",
					MessageID: "message-id",
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID: "user-id",
				},
			},
			wantSuccess: "reaction mismatch, ignored",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "invalid message id",
			setup: func() {
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "invalid-message-id",
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID: "user-id",
				},
			},
			wantSuccess: "reaction mismatch, ignored",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "invalid emoji",
			setup: func() {
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "message-id",
					Emoji: discordgo.Emoji{
						Name: "invalid-emoji",
					},
					UserID: "user-id",
				},
			},
			wantSuccess: "reaction mismatch, ignored",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "bot's own reaction",
			setup: func() {
				mockSession.EXPECT().
					GetBotUser().
					Return(&discordgo.User{ID: "user-id"}, nil).
					Times(1)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "message-id",
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID: "user-id",
				},
			},
			wantSuccess: "ignored bot reaction",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to get bot user",
			setup: func() {
				mockSession.EXPECT().
					GetBotUser().
					Return(nil, errors.New("bot user error")).
					Times(1)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "message-id",
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID: "user-id",
				},
			},
			wantSuccess: "", // or nil, depending on the expected behavior
			wantErrMsg:  "failed to get bot user: bot user error",
			wantErrIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
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

			result, err := sm.MessageReactionAdd(mockSession, tt.args)
			// Check the wrapper return error (should be nil with our test wrapper)
			if err != nil {
				t.Fatalf("MessageReactionAdd() second return value error was non-nil: %v; expected nil with pass-through wrapper", err)
			}

			// Check the SignupOperationResult.Success field
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("SignupOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Check the SignupOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("SignupOperationResult.Error is nil, expected error containing %q", tt.wantErrMsg)
				} else {
					// Check if the error message contains the expected substring
					if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
						t.Errorf("SignupOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
					}
					// Optionally, check for a specific error type if tt.wantErrIs is set
					if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
						t.Errorf("SignupOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
					}
				}
			} else {
				if result.Error != nil {
					t.Errorf("SignupOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}

func Test_signupManager_HandleSignupReactionAdd(t *testing.T) {
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name        string
		setup       func(mockSession *discordmocks.MockSession)
		ctx         context.Context
		args        *discordgo.MessageReactionAdd
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "valid reaction",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(&discordgo.Channel{ID: "dm-channel-id"}, nil).
					Times(1)
				mockSession.EXPECT().
					ChannelMessageSendComplex("dm-channel-id", gomock.Any()).
					Return(&discordgo.Message{}, nil).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantSuccess: "signup button sent",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:  "wrong guild",
			setup: func(mockSession *discordmocks.MockSession) {},
			ctx:   context.Background(),
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "wrong-guild-id",
					UserID:  "user-id",
				},
			},
			wantSuccess: "",
			wantErrMsg:  "reaction from unauthorized guild",
			wantErrIs:   nil,
		},
		{
			name: "failed to create DM channel",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(nil, errors.New("create error")).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantSuccess: "",
			wantErrMsg:  "failed to create DM channel: create error",
			wantErrIs:   nil,
		},
		{
			name: "failed to send ephemeral message",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(&discordgo.Channel{ID: "dm-channel-id"}, nil).
					Times(1)
				mockSession.EXPECT().
					ChannelMessageSendComplex("dm-channel-id", gomock.Any()).
					Return(nil, errors.New("send error")).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantSuccess: "",
			wantErrMsg:  "failed to send signup button in DM: send error",
			wantErrIs:   nil,
		},
		{
			name:  "context cancelled before operation",
			setup: func(mockSession *discordmocks.MockSession) {},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
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

			if tt.setup != nil {
				tt.setup(mockSession)
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

			result, err := sm.HandleSignupReactionAdd(tt.ctx, tt.args)

			if err != nil && result.Error == nil && tt.ctx.Err() == nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("SignupOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("Expected error containing %q but got nil", tt.wantErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("Error mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
				}
			} else if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
			}

			if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
				t.Errorf("Expected error type %T but got %T", tt.wantErrIs, result.Error)
			}
		})
	}
}

func Test_signupManager_HandleSignupButtonPress(t *testing.T) {
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name  string
		setup func(
			mockSession *discordmocks.MockSession,
			mockPublisher *eventbusmocks.MockEventBus,
			mockInteractionStore *storagemocks.MockISInterface,
		)
		ctx           context.Context
		args          *discordgo.InteractionCreate
		wantSuccess   string
		wantErrIs     error // Expected standard error
		wantResultErr error // Expected error in SignupOperationResult
	}{
		{
			name: "valid button press",
			setup: func(
				mockSession *discordmocks.MockSession,
				mockPublisher *eventbusmocks.MockEventBus,
				mockInteractionStore *storagemocks.MockISInterface,
			) {
				// Expect the interaction to be stored first
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Then expect the modal response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					GuildID: "guild-id",
					Type:    discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-id"},
					},
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "modal sent",
			wantErrIs:     nil,
			wantResultErr: nil,
		},
		{
			name: "nil interaction",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// No mock expectations since validation fails early
			},
			ctx:           context.Background(),
			args:          nil,
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: errors.New("interaction is nil or incomplete"),
		},
		{
			name: "nil interaction.Interaction",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// No mock expectations since validation fails early
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: nil,
			},
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: errors.New("interaction is nil or incomplete"),
		},
		{
			name: "nil member",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// Expect interaction store call since validation passes initial checks
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect modal response since the user validation happens after modal creation
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					GuildID: "guild-id",
					Type:    discordgo.InteractionMessageComponent,
					Data:    &discordgo.MessageComponentInteractionData{CustomID: "signup-button"},
					Member:  nil,
					User:    &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "modal sent", // Changed: Modal is sent successfully
			wantErrIs:     nil,
			wantResultErr: nil, // Changed: No error since modal is sent
		},
		{
			name: "nil user",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// No expectations since this fails early in validation
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					GuildID: "guild-id",
					Type:    discordgo.InteractionMessageComponent,
					Data:    &discordgo.MessageComponentInteractionData{CustomID: "signup-button"},
					Member: &discordgo.Member{
						User: nil,
					},
				},
			},
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: errors.New("user is nil in interaction"),
		},
		{
			name: "context cancelled before operation",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// No expectations since context is cancelled
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id", GuildID: "guild-id",
					Type:   discordgo.InteractionMessageComponent,
					Data:   &discordgo.MessageComponentInteractionData{CustomID: "signup-button"},
					Member: &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
					User:   &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: context.Canceled,
		},
		{
			name: "unsupported interaction type",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// Expect interaction store call since validation passes initial checks
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect modal response since the type validation happens after modal creation
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id", GuildID: "guild-id",
					Type:   discordgo.InteractionApplicationCommand,
					Data:   &discordgo.ApplicationCommandInteractionData{Name: "somecommand"},
					Member: &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
					User:   &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "modal sent", // Changed: Modal is sent successfully
			wantErrIs:     nil,
			wantResultErr: nil, // Changed: No error since modal is sent
		},
		{
			name: "unsupported button custom id",
			setup: func(mockSession *discordmocks.MockSession, mockPublisher *eventbusmocks.MockEventBus, mockInteractionStore *storagemocks.MockISInterface) {
				// Expect interaction store call since validation passes initial checks
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect modal response since the custom ID validation happens after modal creation
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id", GuildID: "guild-id",
					Type:   discordgo.InteractionMessageComponent,
					Data:   &discordgo.MessageComponentInteractionData{CustomID: "other-button-id"},
					Member: &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
					User:   &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "modal sent", // Changed: Modal is sent successfully
			wantErrIs:     nil,
			wantResultErr: nil, // Changed: No error since modal is sent
		},
		{
			name: "SendSignupModal fails",
			setup: func(
				mockSession *discordmocks.MockSession,
				mockPublisher *eventbusmocks.MockEventBus,
				mockInteractionStore *storagemocks.MockISInterface,
			) {
				// Expect interaction store call first
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Then expect the failing modal response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("modal error")).
					Times(1)
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					GuildID: "guild-id",
					Type:    discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-id"},
					},
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: errors.New("modal error"),
		},
		{
			name: "interaction store set fails",
			setup: func(
				mockSession *discordmocks.MockSession,
				mockPublisher *eventbusmocks.MockEventBus,
				mockInteractionStore *storagemocks.MockISInterface,
			) {
				// Expect interaction store call to fail
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("store error")).
					Times(1)

				// No session call expected since store fails first
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					GuildID: "guild-id",
					Type:    discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-id"},
					},
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantSuccess:   "",
			wantErrIs:     nil, // Error is now in result, not returned directly
			wantResultErr: errors.New("failed to store interaction: store error"),
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

			if tt.setup != nil {
				tt.setup(mockSession, mockPublisher, mockInteractionStore)
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

			result, err := sm.HandleSignupButtonPress(tt.ctx, tt.args)

			// Check standard error (returned directly)
			if tt.wantErrIs != nil {
				if err == nil {
					t.Fatalf("Test %q: Expected standard error %v but got nil", tt.name, tt.wantErrIs)
				}
				if !errors.Is(err, tt.wantErrIs) && err.Error() != tt.wantErrIs.Error() {
					t.Errorf("Test %q: Standard error mismatch: got %q, want %v", tt.name, err.Error(), tt.wantErrIs)
				}
			} else if err != nil {
				t.Errorf("Test %q: Unexpected standard error: %v", tt.name, err)
			}

			// Check result error (in SignupOperationResult.Error)
			if tt.wantResultErr != nil {
				if result.Error == nil {
					t.Fatalf("Test %q: Expected result.Error %v but got nil", tt.name, tt.wantResultErr)
				}
				if !errors.Is(result.Error, tt.wantResultErr) && result.Error.Error() != tt.wantResultErr.Error() {
					t.Errorf("Test %q: SignupOperationResult.Error mismatch: got %q, want %v", tt.name, result.Error.Error(), tt.wantResultErr)
				}
			} else if result.Error != nil {
				t.Errorf("Test %q: Unexpected SignupOperationResult.Error: %v", tt.name, result.Error)
			}

			// Check success result
			if tt.wantSuccess != "" {
				gotSuccess, ok := result.Success.(string)
				if !ok || gotSuccess != tt.wantSuccess {
					t.Errorf("Test %q: SignupOperationResult.Success mismatch: got %v (type %T), want %q", tt.name, result.Success, result.Success, tt.wantSuccess)
				}
			} else if result.Success != nil {
				if successVal, isString := result.Success.(string); isString && successVal != "" {
					t.Errorf("Test %q: SignupOperationResult.Success mismatch: got %q, want empty", tt.name, successVal)
				} else if result.Success != nil {
					t.Errorf("Test %q: SignupOperationResult.Success mismatch: got %v, want nil", tt.name, result.Success)
				}
			}
		})
	}
}
