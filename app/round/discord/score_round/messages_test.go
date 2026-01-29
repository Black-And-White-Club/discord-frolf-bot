package scoreround

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func Test_scoreRoundManager_SendScoreUpdateConfirmation(t *testing.T) {
	fakeSession := discord.NewFakeSession()
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
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					score := sharedtypes.Score(42)
					expectedMsg := fmt.Sprintf("<@%s> Your score of %d has been recorded!", "user-123", score)
					if channelID != "channel-123" {
						t.Errorf("expected channel-123, got %s", channelID)
					}
					if data.Content != expectedMsg {
						t.Errorf("expected content %q, got %q", expectedMsg, data.Content)
					}
					return &discordgo.Message{}, nil
				}
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
				fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("discord api error")
				}
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
				session: fakeSession,
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
	fakeSession := discord.NewFakeSession()
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
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					if recipientID != "user-123" {
						t.Errorf("expected recipient user-123, got %s", recipientID)
					}
					return &discordgo.Channel{ID: "dm-channel-123"}, nil
				}

				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					expectedMsg := "We encountered an error updating your score: Invalid score format"
					if channelID != "dm-channel-123" {
						t.Errorf("expected channel dm-channel-123, got %s", channelID)
					}
					if content != expectedMsg {
						t.Errorf("expected content %q, got %q", expectedMsg, content)
					}
					return &discordgo.Message{}, nil
				}
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
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("failed to create DM channel")
				}
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
				fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "dm-channel-789"}, nil
				}

				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("failed to send DM")
				}
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
				session: fakeSession,
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
