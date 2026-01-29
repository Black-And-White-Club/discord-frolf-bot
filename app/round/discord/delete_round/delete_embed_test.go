package deleteround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
)

func Test_deleteRoundManager_DeleteRoundEventEmbed(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(f *discord.FakeSession, eventMessageID string)
		channelID       string
		eventMessageID  string
		expectedErr     bool
		expectedError   string
		expectedSuccess bool
	}{
		{
			name: "successful delete",
			setup: func(f *discord.FakeSession, eventMessageID string) {
				f.ChannelMessageDeleteFunc = func(channelID, messageID string, options ...discordgo.RequestOption) error {
					if channelID != "channel-123" || messageID != eventMessageID {
						t.Errorf("Expected ChannelMessageDelete with channel-123 and %s, got %s and %s", eventMessageID, channelID, messageID)
					}
					return nil
				}
			},
			channelID:       "channel-123",
			eventMessageID:  "12345",
			expectedErr:     false,
			expectedSuccess: true,
		},
		{
			name: "missing channel ID",
			setup: func(f *discord.FakeSession, eventMessageID string) {
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
			setup: func(f *discord.FakeSession, eventMessageID string) {
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
			setup: func(f *discord.FakeSession, eventMessageID string) {
				f.ChannelMessageDeleteFunc = func(channelID, messageID string, options ...discordgo.RequestOption) error {
					return errors.New("message not found")
				}
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
			fakeSession := discord.NewFakeSession()
			tt.setup(fakeSession, tt.eventMessageID)

			dem := &deleteRoundManager{
				session: fakeSession,
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
