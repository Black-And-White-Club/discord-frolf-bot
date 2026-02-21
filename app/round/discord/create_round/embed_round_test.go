package createround

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_createRoundManager_SendRoundEventEmbed(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(f *discord.FakeSession)
		expectedErr   bool
		expectedError string
	}{
		{
			name: "successful embed creation",
			setup: func(f *discord.FakeSession) {
				f.UserFunc = func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return &discordgo.User{ID: "user-123", Username: "TestUser"}, nil
				}
				f.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{Nick: "NickName"}, nil
				}
				f.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "msg-1"}, nil
				}
			},
			expectedErr: false,
		},
		{
			name: "user fetch fails",
			setup: func(f *discord.FakeSession) {
				f.UserFunc = func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return nil, errors.New("user fetch fail")
				}
			},
			expectedErr:   true,
			expectedError: "failed to get creator info: user fetch fail",
		},
		{
			name: "nickname fallback (member not found)",
			setup: func(f *discord.FakeSession) {
				f.UserFunc = func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return &discordgo.User{ID: "user-123", Username: "FallbackUser"}, nil
				}
				f.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return nil, errors.New("no member")
				}
				f.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "msg-2"}, nil
				}
			},
			expectedErr: false,
		},
		{
			name: "message send failure",
			setup: func(f *discord.FakeSession) {
				f.UserFunc = func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return &discordgo.User{ID: "user-123", Username: "TestUser"}, nil
				}
				f.GuildMemberFunc = func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{Nick: "NickName"}, nil
				}
				f.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("send fail")
				}
			},
			expectedErr:   true,
			expectedError: "failed to send embed message: send fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()

			if tt.setup != nil {
				tt.setup(fakeSession)
			}

			manager := &createRoundManager{
				session: fakeSession,
				config: &config.Config{
					Discord: config.DiscordConfig{
						GuildID: "guild-id",
					},
				},
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return fn(ctx)
				},
				logger: slog.Default(),
			}

			result, err := manager.SendRoundEventEmbed(
				"guild-id",
				"channel-123",
				roundtypes.Title("Round Title"),
				roundtypes.Description("This is a description"),
				sharedtypes.StartTime(time.Date(2025, 3, 14, 15, 0, 0, 0, time.UTC)),
				roundtypes.Location("Test Park"),
				sharedtypes.DiscordID("user-123"),
				sharedtypes.RoundID(uuid.New()),
			)

			if tt.expectedErr {
				if err == nil && result.Error == nil {
					t.Errorf("%s: Expected error, got none (err: nil, result.Error: nil)", tt.name)
				}
				if tt.expectedError != "" {
					var actualError string
					if err != nil {
						actualError = err.Error()
					} else if result.Error != nil {
						actualError = result.Error.Error()
					}
					if !strings.Contains(actualError, tt.expectedError) {
						t.Errorf("%s: Expected error containing: %v, got: %v", tt.name, tt.expectedError, actualError)
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.name, err)
				}
				if result.Error != nil {
					t.Errorf("%s: Unexpected result.Error: %v", tt.name, result.Error)
				}
				if result.Success == nil {
					t.Errorf("%s: Expected success message, got nil", tt.name)
				}
			}
		})
	}
}

func Test_createRoundManager_SendRoundEventURL(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(f *discord.FakeSession)
		expectedErr   bool
		expectedError string
	}{
		{
			name: "successful send",
			setup: func(f *discord.FakeSession) {
				f.ChannelMessageSendFunc = func(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					expectedURL := "https://discord.com/events/guild-id/event-123"
					if content != expectedURL {
						t.Errorf("expected content to be %q, got %q", expectedURL, content)
					}
					return &discordgo.Message{ID: "msg-1"}, nil
				}
			},
			expectedErr: false,
		},
		{
			name: "send failure",
			setup: func(f *discord.FakeSession) {
				f.ChannelMessageSendFunc = func(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("send failed")
				}
			},
			expectedErr:   true,
			expectedError: "failed to send event url message: send failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()

			if tt.setup != nil {
				tt.setup(fakeSession)
			}

			manager := &createRoundManager{
				session: fakeSession,
				config: &config.Config{
					Discord: config.DiscordConfig{
						GuildID: "guild-id",
					},
				},
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return fn(ctx)
				},
				logger: slog.Default(),
			}

			result, err := manager.SendRoundEventURL(
				"guild-id",
				"channel-123",
				"event-123",
			)

			if tt.expectedErr {
				if err == nil && result.Error == nil {
					t.Errorf("%s: Expected error, got none", tt.name)
				}
				if tt.expectedError != "" {
					var actualError string
					if err != nil {
						actualError = err.Error()
					} else if result.Error != nil {
						actualError = result.Error.Error()
					}
					if !strings.Contains(actualError, tt.expectedError) {
						t.Errorf("%s: Expected error containing: %v, got: %v", tt.name, tt.expectedError, actualError)
					}
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.name, err)
				}
				if result.Error != nil {
					t.Errorf("%s: Unexpected result.Error: %v", tt.name, result.Error)
				}
				if result.Success == nil {
					t.Errorf("%s: Expected success message, got nil", tt.name)
				}
			}
		})
	}
}
