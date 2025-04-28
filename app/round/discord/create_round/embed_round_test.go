package createround

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_createRoundManager_SendRoundEventEmbed(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(mockSession *discordmocks.MockSession)
		expectedErr   bool
		expectedError string
	}{
		{
			name: "successful embed creation",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{ID: "user-123", Username: "TestUser"}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: "NickName"}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageSendComplex("channel-123", gomock.Any()).
					Return(&discordgo.Message{ID: "msg-1"}, nil).
					Times(1)
			},
			expectedErr: false,
		},
		{
			name: "user fetch fails",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User("user-123").
					Return(nil, errors.New("user fetch fail")).
					Times(1)
			},
			expectedErr:   true,
			expectedError: "failed to get creator info: user fetch fail",
		},
		{
			name: "nickname fallback (member not found)",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{ID: "user-123", Username: "FallbackUser"}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(nil, errors.New("no member")).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageSendComplex("channel-123", gomock.Any()).
					Return(&discordgo.Message{ID: "msg-2"}, nil).
					Times(1)
			},
			expectedErr: false,
		},
		{
			name: "message send failure",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{ID: "user-123", Username: "TestUser"}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: "NickName"}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageSendComplex("channel-123", gomock.Any()).
					Return(nil, errors.New("send fail")).
					Times(1)
			},
			expectedErr:   true,
			expectedError: "failed to send embed message: send fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)

			if tt.setup != nil {
				tt.setup(mockSession)
			}

			manager := &createRoundManager{
				session: mockSession,
				config: &config.Config{
					Discord: config.DiscordConfig{
						GuildID: "guild-id",
					},
				},
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			result, err := manager.SendRoundEventEmbed(
				"channel-123",
				"Round Title",
				"This is a description",
				sharedtypes.StartTime(time.Date(2025, 3, 14, 15, 0, 0, 0, time.UTC)),
				"Test Park",
				"user-123",
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
