package scoreround

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_scoreRoundManager_SendScoreUpdateConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{}

	type args struct {
		channelID string
		userID    sharedtypes.DiscordID
		score     sharedtypes.Score
	}

	tests := []struct {
		name    string
		setup   func()
		args    args
		wantErr bool
	}{
		{
			name: "successfully sends score update confirmation",
			setup: func() {
				score := sharedtypes.Score(42)
				expectedMsg := fmt.Sprintf("<@%s> Your score of %d has been recorded!", "user-123", score)

				mockSession.EXPECT().ChannelMessageSendComplex(
					gomock.Eq("channel-123"),
					gomock.Eq(&discordgo.MessageSend{
						Content: expectedMsg,
						AllowedMentions: &discordgo.MessageAllowedMentions{
							Users: []string{"user-123"},
						},
					}),
				).Return(&discordgo.Message{}, nil).Times(1)
			},
			args: args{
				channelID: "channel-123",
				userID:    "user-123",
				score:     sharedtypes.Score(42),
			},
			wantErr: false,
		},
		{
			name: "handles error when sending message fails",
			setup: func() {
				score := sharedtypes.Score(75)
				expectedMsg := fmt.Sprintf("<@%s> Your score of %d has been recorded!", "user-456", score)

				mockSession.EXPECT().ChannelMessageSendComplex(
					gomock.Eq("channel-456"),
					gomock.Eq(&discordgo.MessageSend{
						Content: expectedMsg,
						AllowedMentions: &discordgo.MessageAllowedMentions{
							Users: []string{"user-456"},
						},
					}),
				).Return(nil, errors.New("discord api error")).Times(1)
			},
			args: args{
				channelID: "channel-456",
				userID:    "user-456",
				score:     sharedtypes.Score(75),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			srm := &scoreRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			srm.operationWrapper = func(ctx context.Context, name string, fn func(context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
				return fn(ctx)
			}

			_, err := srm.SendScoreUpdateConfirmation(context.Background(), tt.args.channelID, tt.args.userID, &tt.args.score)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendScoreUpdateConfirmation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_scoreRoundManager_SendScoreUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{}

	type args struct {
		userID       sharedtypes.DiscordID
		errorMessage string
	}

	tests := []struct {
		name    string
		setup   func()
		args    args
		wantErr bool
	}{
		{
			name: "successfully sends error message in DM",
			setup: func() {
				expectedMsg := "We encountered an error updating your score: Invalid score format"

				mockSession.EXPECT().
					UserChannelCreate("user-123").
					Return(&discordgo.Channel{ID: "dm-channel-123"}, nil).
					Times(1)

				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("dm-channel-123"),
					gomock.Eq(expectedMsg),
				).Return(&discordgo.Message{}, nil).Times(1)
			},
			args: args{
				userID:       "user-123",
				errorMessage: "Invalid score format",
			},
			wantErr: false,
		},
		{
			name: "handles error when DM channel creation fails",
			setup: func() {
				mockSession.EXPECT().
					UserChannelCreate("user-456").
					Return(nil, errors.New("failed to create DM channel")).
					Times(1)
			},
			args: args{
				userID:       "user-456",
				errorMessage: "Score out of range",
			},
			wantErr: true,
		},
		{
			name: "handles error when sending DM message fails",
			setup: func() {
				expectedMsg := "We encountered an error updating your score: Invalid input"

				mockSession.EXPECT().
					UserChannelCreate("user-789").
					Return(&discordgo.Channel{ID: "dm-channel-789"}, nil).
					Times(1)

				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("dm-channel-789"),
					gomock.Eq(expectedMsg),
				).Return(nil, errors.New("failed to send DM")).Times(1)
			},
			args: args{
				userID:       "user-789",
				errorMessage: "Invalid input",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			srm := &scoreRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			srm.operationWrapper = func(ctx context.Context, name string, fn func(context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
				return fn(ctx)
			}

			_, err := srm.SendScoreUpdateError(context.Background(), tt.args.userID, tt.args.errorMessage)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
