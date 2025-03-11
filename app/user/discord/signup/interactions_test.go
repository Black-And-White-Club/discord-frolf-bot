package signup

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_signupManager_MessageReactionAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			SignupChannelID: "channel-id",
			SignupMessageID: "message-id",
			SignupEmoji:     "emoji",
		},
	}

	tests := []struct {
		name    string
		setup   func()
		args    *discordgo.MessageReactionAdd
		wantErr bool
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
					UserID: "user-id",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid channel id",
			setup: func() {
				mockSession.EXPECT().
					GetBotUser().
					Return(&discordgo.User{ID: "bot-user-id"}, nil).
					Times(1)
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
		},
		{
			name: "bot's own reaction",
			setup: func() {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(nil, errors.New("test error")).
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
			wantErr: true,
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
				logger:           mockLogger,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			sm.MessageReactionAdd(mockSession, tt.args)
		})
	}
}

func Test_signupManager_HandleSignupReactionAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name    string
		setup   func()
		args    *discordgo.MessageReactionAdd
		wantErr bool
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
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantErr: false,
		},
		{
			name: "wrong guild",
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "wrong-guild-id",
					UserID:  "user-id",
				},
			},
			wantErr: true,
		},
		{
			name: "failed to create DM channel",
			setup: func() {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(nil, errors.New("create error")).
					Times(1)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantErr: true,
		},
		{
			name: "failed to send ephemeral message",
			setup: func() {
				mockSession.EXPECT().
					UserChannelCreate(gomock.Any()).
					Return(&discordgo.Channel{}, nil).
					Times(1)
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("send error")).
					Times(1)
			},
			args: &discordgo.MessageReactionAdd{
				MessageReaction: &discordgo.MessageReaction{
					GuildID: "guild-id",
					UserID:  "user-id",
				},
			},
			wantErr: true,
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
				logger:           mockLogger,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			sm.HandleSignupReactionAdd(context.Background(), tt.args)
		})
	}
}

func Test_signupManager_HandleSignupButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	tests := []struct {
		name    string
		setup   func()
		args    *discordgo.InteractionCreate
		wantErr bool
	}{
		{
			name: "valid interaction",
			setup: func() {

			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid interaction",
			setup: func() {

			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "invalid-button",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to send signup modal",
			setup: func() {
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to send ephemeral message",
			setup: func() {
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionMessageComponent,
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "signup-button",
					},
				},
			},
			wantErr: true,
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
				logger:           mockLogger,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}
			sm.HandleSignupButtonPress(context.Background(), tt.args)
		})
	}
}
