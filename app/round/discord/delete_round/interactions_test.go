package deleteround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func Test_deleteRoundManager_HandleDeleteRoundButton(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeConfig := &config.Config{}
	metrics := &discordmetrics.NoOpMetrics{}
	logger := loggerfrolfbot.NoOpLogger

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
				Message: &discordgo.Message{
					ID: "message-123",
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
		expectErr     bool
		expectedError string
		expectSuccess bool
	}{
		{
			name: "successful delete request",
			setup: func() {
				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return &message.Message{UUID: "msg-456"}, nil
				}

				fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
					return nil
				}

				fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "message-456"}, nil
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"), // Use valid UUID string
			},
			expectSuccess: true,
			expectErr:     false,
		},
		{
			name: "invalid custom ID format",
			setup: func() {
				// No fake expectations for this test case
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete"), // Missing the round ID part
			},
			expectErr:     true,
			expectedError: "invalid custom_id format",
		},
		{
			name: "invalid round ID",
			setup: func() {
				fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "message-456"}, nil
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|invalid"),
			},
			expectErr:     true,
			expectedError: "failed to parse round ID as UUID",
		},
		{
			name: "interaction respond error",
			setup: func() {
				// Fake InteractionRespond call with error
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to respond to interaction")
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to acknowledge interaction",
		},
		{
			name: "create result message error",
			setup: func() {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return nil, errors.New("failed to create result message")
				}

				fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "message-456"}, nil
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to create result message",
		},
		{
			name: "publish error",
			setup: func() {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return &message.Message{UUID: "msg-456"}, nil
				}

				fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
					return errors.New("failed to publish message")
				}

				fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "message-456"}, nil
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to publish delete request",
		},
		{
			name: "followup message error",
			setup: func() {
				fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}

				fakeHelper.CreateResultMessageFunc = func(originalMsg *message.Message, payload interface{}, topic string) (*message.Message, error) {
					return &message.Message{UUID: "msg-456"}, nil
				}

				fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
					return nil
				}

				fakeSession.FollowupMessageCreateFunc = func(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("failed to send followup message")
				}
			},
			args: struct {
				ctx context.Context
				i   *discordgo.InteractionCreate
			}{
				ctx: context.Background(),
				i:   createInteraction("round_delete|00000000-0000-0000-0000-0000000003b1"),
			},
			expectErr:     true,
			expectedError: "failed to send followup message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			drm := &deleteRoundManager{
				session:   fakeSession,
				publisher: fakePublisher,
				logger:    logger,
				helper:    fakeHelper,
				config:    fakeConfig,
				operationWrapper: func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (DeleteRoundOperationResult, error)) (DeleteRoundOperationResult, error) {
					return operationFunc(ctx)
				},
				metrics: metrics,
			}

			result, err := drm.HandleDeleteRoundButton(tt.args.ctx, tt.args.i)

			if tt.expectErr {
				if err == nil && result.Error == nil {
					t.Errorf("%s: Expected error, got nil (err and result.Error)", tt.name)
				}
				if tt.expectedError != "" {
					var actualError string
					if err != nil {
						actualError = err.Error()
					} else if result.Error != nil {
						actualError = result.Error.Error()
					}
					if !strings.Contains(actualError, tt.expectedError) {
						t.Errorf("%s: Expected error containing: %q, got: %q", tt.name, tt.expectedError, actualError)
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.name, err)
				}
				if result.Error != nil {
					t.Errorf("%s: Unexpected result.Error: %v", tt.name, result.Error)
				}
			}
			if tt.expectSuccess != (result.Success != nil) {
				t.Errorf("%s: Expected success: %v, got %v", tt.name, tt.expectSuccess, result.Success != nil)
			}
		})
	}
}
