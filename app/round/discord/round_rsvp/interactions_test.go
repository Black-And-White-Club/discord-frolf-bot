package roundrsvp

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_roundRsvpManager_HandleRoundResponse(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	mockLogger := loggerfrolfbot.NoOpLogger
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
				Message: &discordgo.Message{
					ID: "message-123",
				},
			},
		}
	}

	tests := []struct {
		name  string
		setup func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers)
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
		expectedError string
	}{
		{
			name: "interaction respond error",
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to respond to interaction")
				}
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return nil, errors.New("failed to create result message")
				}
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return &message.Message{UUID: "msg-123"}, nil
				}

				fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
					return errors.New("failed to publish message")
				}
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
			fakeSession := discord.NewFakeSession()
			fakePublisher := &testutils.FakeEventBus{}
			fakeHelper := &testutils.FakeHelpers{}

			if tt.setup != nil {
				tt.setup(fakeSession, fakePublisher, fakeHelper)
			}

			rrm := &roundRsvpManager{
				session:   fakeSession,
				publisher: fakePublisher,
				logger:    mockLogger,
				helper:    fakeHelper,
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
	testRoundID := sharedtypes.RoundID(uuid.New())
	mockLogger := loggerfrolfbot.NoOpLogger
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
				Message: &discordgo.Message{
					ID: "message-123",
				},
			},
		}
	}

	tests := []struct {
		name  string
		setup func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers)
		args  struct {
			ctx context.Context
			i   *discordgo.InteractionCreate
		}
		expectedError string
	}{
		{
			name: "invalid custom ID format",
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: "message-123",
						Embeds: []*discordgo.MessageEmbed{
							{Title: "Test Round", Description: "Test Description"},
						},
					}, nil
				}

				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to respond")
				}
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: "message-123",
						Embeds: []*discordgo.MessageEmbed{
							{Title: "Test Round", Description: "Test Description"},
						},
					}, nil
				}

				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return &message.Message{UUID: "msg-123"}, nil
				}

				fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
					return errors.New("failed to publish")
				}
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
			setup: func(fakeSession *discord.FakeSession, fakePublisher *testutils.FakeEventBus, fakeHelper *testutils.FakeHelpers) {
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
			fakeSession := discord.NewFakeSession()
			fakePublisher := &testutils.FakeEventBus{}
			fakeHelper := &testutils.FakeHelpers{}

			if tt.setup != nil {
				tt.setup(fakeSession, fakePublisher, fakeHelper)
			}

			rrm := &roundRsvpManager{
				session:   fakeSession,
				publisher: fakePublisher,
				logger:    mockLogger,
				helper:    fakeHelper,
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
