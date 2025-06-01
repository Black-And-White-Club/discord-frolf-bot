package deleteround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.uber.org/mock/gomock"
)

func Test_deleteRoundManager_DeleteRoundEventEmbed(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(mockSession *discordmocks.MockSession, eventMessageID string)
		channelID       string
		eventMessageID  string
		expectedErr     bool
		expectedError   string
		expectedSuccess bool
	}{
		{
			name: "successful delete",
			setup: func(mockSession *discordmocks.MockSession, eventMessageID string) {
				mockSession.EXPECT().
					ChannelMessageDelete("channel-123", eventMessageID).
					Return(nil).
					Times(1)
			},
			channelID:       "channel-123",
			eventMessageID:  "12345",
			expectedErr:     false,
			expectedSuccess: true,
		},
		{
			name: "missing channel ID",
			setup: func(mockSession *discordmocks.MockSession, eventMessageID string) {
				// No expectations, function should return before calling discordgo
			},
			channelID:       "",
			eventMessageID:  "12345",
			expectedErr:     true,
			expectedError:   "channelID or discordMessageID is missing",
			expectedSuccess: false,
		},
		{
			name: "missing event message ID",
			setup: func(mockSession *discordmocks.MockSession, eventMessageID string) {
				// No expectations, function should return before calling discordgo
			},
			channelID:       "channel-123",
			eventMessageID:  "",
			expectedErr:     true,
			expectedError:   "channelID or discordMessageID is missing",
			expectedSuccess: false,
		},
		{
			name: "failed to delete message",
			setup: func(mockSession *discordmocks.MockSession, eventMessageID string) {
				mockSession.EXPECT().
					ChannelMessageDelete("channel-123", eventMessageID).
					Return(errors.New("message not found")).
					Times(1)
			},
			channelID:       "channel-123",
			eventMessageID:  "12345",
			expectedErr:     true,
			expectedError:   "failed to delete message",
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			tt.setup(mockSession, tt.eventMessageID)

			dem := &deleteRoundManager{
				session: mockSession,
				logger:  loggerfrolfbot.NoOpLogger,
				config: &config.Config{
					Discord: config.DiscordConfig{
						GuildID: "guild-id",
					},
				},
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (DeleteRoundOperationResult, error)) (DeleteRoundOperationResult, error) {
					return fn(ctx) // Bypass the operationWrapper for simpler testing
				},
			}

			result, err := dem.DeleteRoundEventEmbed(context.Background(), tt.eventMessageID, tt.channelID)

			if tt.expectedErr {
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
				if result.Success == true {
					t.Errorf("%s: Expected Success to be false, got true", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.name, err)
				}
				if result.Error != nil {
					t.Errorf("%s: Unexpected result.Error: %v", tt.name, result.Error)
				}
				if result.Success != tt.expectedSuccess {
					t.Errorf("%s: Expected Success: %v, got %v", tt.name, tt.expectedSuccess, result.Success)
				}
			}
		})
	}
}
