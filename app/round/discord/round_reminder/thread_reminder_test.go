package roundreminder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_roundReminderManager_SendRoundReminder(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	mockLogger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	sampleTime := time.Now()
	sampleLocation := "Test Location"
	testRoundID := uuid.New()

	samplePayload := &roundevents.DiscordReminderPayloadV1{
		RoundID:          sharedtypes.RoundID(testRoundID),
		RoundTitle:       "Test Round",
		UserIDs:          []sharedtypes.DiscordID{"user-123", "user-456"},
		ReminderType:     "1-hour",
		DiscordChannelID: "channel-123",
		DiscordGuildID:   "guild-id",
		EventMessageID:   "12345",
		StartTime:        (*sharedtypes.StartTime)(&sampleTime),
		Location:         (roundtypes.Location)(sampleLocation),
	}

	tests := []struct {
		name    string
		setup   func()
		payload *roundevents.DiscordReminderPayloadV1
		want    RoundReminderOperationResult
		wantErr bool
	}{
		{
			name: "successful reminder with new thread",
			setup: func() {
				fakeSession.GetChannelFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
				fakeSession.ThreadsActiveFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
					return &discordgo.ThreadsList{}, nil
				}
				fakeSession.MessageThreadStartComplexFunc = func(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "thread-123"}, nil
				}
				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
		},
		{
			name: "failed to get channel",
			setup: func() {
				fakeSession.GetChannelFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("channel not found")
				}
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to get channel")},
			wantErr: true,
		},
		{
			name: "thread creation fails and falls back to main channel",
			setup: func() {
				fakeSession.GetChannelFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
				fakeSession.ThreadsActiveFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
					return &discordgo.ThreadsList{}, nil
				}
				fakeSession.MessageThreadStartComplexFunc = func(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("thread create failed")
				}
				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
		},
		{
			name: "send to thread fails",
			setup: func() {
				fakeSession.GetChannelFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
				fakeSession.ThreadsActiveFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
					return &discordgo.ThreadsList{}, nil
				}
				fakeSession.MessageThreadStartComplexFunc = func(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "thread-123"}, nil
				}
				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("send failed")
				}
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to send reminder message")},
			wantErr: true,
		},
		{
			name: "thread already exists via race condition",
			setup: func() {
				fakeSession.GetChannelFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{}, nil
				}
				firstMessageCall := true
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					if firstMessageCall {
						firstMessageCall = false
						return &discordgo.Message{ID: "12345"}, nil
					}
					return &discordgo.Message{
						Thread: &discordgo.Channel{ID: "thread-123"},
					}, nil
				}
				fakeSession.ThreadsActiveFunc = func(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
					return &discordgo.ThreadsList{}, nil
				}
				fakeSession.MessageThreadStartComplexFunc = func(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("Thread already exists")
				}
				fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
		},
		{
			name: "missing event message id",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:          sharedtypes.RoundID(testRoundID),
				RoundTitle:       "Test Round",
				EventMessageID:   "",
				DiscordChannelID: "channel-123",
			},
			want:    RoundReminderOperationResult{Error: errors.New("no message ID provided in payload")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roundReminderManager{
				session: fakeSession,
				config:  mockConfig,
				logger:  mockLogger,
				operationWrapper: func(ctx context.Context, _ string,
					fn func(ctx context.Context) (RoundReminderOperationResult, error),
				) (RoundReminderOperationResult, error) {
					return fn(ctx)
				},
			}

			got, err := rm.SendRoundReminder(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.want.Error != nil &&
				!strings.Contains(err.Error(), tt.want.Error.Error()) {
				t.Fatalf("error mismatch: got %v want %v", err, tt.want.Error)
			}

			if got.Success != tt.want.Success {
				t.Fatalf("success = %v want %v", got.Success, tt.want.Success)
			}
		})
	}
}
