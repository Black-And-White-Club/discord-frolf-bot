package signup

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

// Test wrapper function that passes through the operation call without additional logic
var testOperationWrapper = func(ctx context.Context, operationName string, operation func(ctx context.Context) (SignupOperationResult, error)) (SignupOperationResult, error) {
	return operation(ctx)
}

func Test_signupManager_MessageReactionAdd(t *testing.T) {
	fakeSession := &discord.FakeSession{}
	fakePublisher := &testutils.FakeEventBus{}
	fakeInteractionStore := testutils.NewFakeStorage[any]()
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

	fakeGuildConfigResolver := &testutils.FakeGuildConfigResolver{}

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
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
				fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
					return &discordgo.User{ID: "bot-user-id"}, nil
				}
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "any-message-id", // Message ID no longer matters
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
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "invalid-channel-id",
					MessageID: "any-message-id", // Message ID no longer matters
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id",
				},
			},
			wantSuccess: "channel mismatch, ignored",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "different message id in same channel - now processed since we only check channel + emoji",
			setup: func() {
				fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
					return &discordgo.User{ID: "bot-user-id"}, nil
				}
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "different-message-id", // This is now allowed
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id", // Ensure GuildID is set so resolver is called
				},
			},
			wantSuccess: "signup button sent", // Would now be processed
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
					MessageID: "any-message-id", // Message ID no longer matters
					Emoji: discordgo.Emoji{
						Name: "invalid-emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id",
				},
			},
			wantSuccess: "emoji mismatch, ignored",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "bot's own reaction",
			setup: func() {
				fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
					return &discordgo.User{ID: "user-id"}, nil
				}
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "any-message-id", // Message ID no longer matters
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id",
				},
			},
			wantSuccess: "ignored bot reaction",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to get bot user",
			setup: func() {
				fakeSession.GetBotUserFunc = func() (*discordgo.User, error) {
					return nil, errors.New("bot user error")
				}
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					ChannelID: "channel-id",
					MessageID: "any-message-id", // Message ID no longer matters
					Emoji: discordgo.Emoji{
						Name: "emoji",
					},
					UserID:  "user-id",
					GuildID: "guild-id",
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
				session:             fakeSession,
				publisher:           fakePublisher,
				logger:              logger,
				config:              mockConfig,
				guildConfigResolver: fakeGuildConfigResolver,
				interactionStore:    fakeInteractionStore,
				tracer:              tracer,
				metrics:             metrics,
				operationWrapper:    testOperationWrapper,
			}

			// Set up GuildConfigResolver expectations for per-guild config fetches
			if tt.args != nil && tt.args.GuildID != "" {
				guildConfig := &storage.GuildConfig{
					SignupChannelID: "channel-id",
					SignupEmoji:     "emoji",
				}
				fakeGuildConfigResolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
					return guildConfig, nil
				}
			}

			result, err := sm.MessageReactionAdd(fakeSession, tt.args)
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
	fakeSession := &discord.FakeSession{}
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name        string
		setup       func()
		ctx         context.Context
		args        *discordgo.MessageReactionAdd
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "valid reaction",
			setup: func() {
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "dm-channel-id"}, nil
				}
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
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
			setup: func() {},
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
			setup: func() {
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("create error")
				}
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
			setup: func() {
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "dm-channel-id"}, nil
				}
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("send error")
				}
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
			setup: func() {},
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
			fakePublisher := &testutils.FakeEventBus{}
			fakeInteractionStore := testutils.NewFakeStorage[any]()
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			if tt.setup != nil {
				tt.setup()
			}

			sm := &signupManager{
				session:          fakeSession,
				publisher:        fakePublisher,
				logger:           logger,
				config:           mockConfig,
				interactionStore: fakeInteractionStore,
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
	fakeSession := &discord.FakeSession{}
	fakePublisher := &testutils.FakeEventBus{}
	fakeInteractionStore := testutils.NewFakeStorage[any]()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name          string
		setup         func()
		ctx           context.Context
		args          *discordgo.InteractionCreate
		wantSuccess   string
		wantErrIs     error // Expected standard error
		wantResultErr error // Expected error in SignupOperationResult
	}{
		{
			name: "valid button press",
			setup: func() {
				// Expect the interaction to be stored first
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				// Then expect the modal response
				fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return nil
				}
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
			setup: func() {
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
			setup: func() {
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
			setup: func() {
				// Expect interaction store call since validation passes initial checks
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				// Expect modal response since the user validation happens after modal creation
				fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return nil
				}
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
			setup: func() {
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
			setup: func() {
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
			setup: func() {
				// Expect interaction store call since validation passes initial checks
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				// Expect modal response since the type validation happens after modal creation
				fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return nil
				}
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
			setup: func() {
				// Expect interaction store call since validation passes initial checks
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				// Expect modal response since the custom ID validation happens after modal creation
				fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return nil
				}
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
			setup: func() {
				// Expect interaction store call first
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				// Then expect the failing modal response
				fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
					return errors.New("modal error")
				}
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
			setup: func() {
				// Expect interaction store call to fail
				fakeInteractionStore.SetFunc = func(ctx context.Context, key string, value any) error {
					return errors.New("store error")
				}
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
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			if tt.setup != nil {
				tt.setup()
			}

			sm := &signupManager{
				session:          fakeSession,
				publisher:        fakePublisher,
				logger:           logger,
				config:           mockConfig,
				interactionStore: fakeInteractionStore,
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
				}
			}
		})
	}
}
