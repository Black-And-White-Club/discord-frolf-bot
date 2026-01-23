package roundreminder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_roundReminderManager_SendRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
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
				mockSession.EXPECT().GetChannel("channel-123").Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage("channel-123", "12345").
					Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive("channel-123").
					Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().
					MessageThreadStartComplex("channel-123", "12345", gomock.Any()).
					Return(&discordgo.Channel{ID: "thread-123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend("thread-123", gomock.Any()).
					Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
		},
		{
			name: "failed to get channel",
			setup: func() {
				mockSession.EXPECT().
					GetChannel("channel-123").
					Return(nil, errors.New("channel not found"))
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to get channel")},
			wantErr: true,
		},
		{
			name: "thread creation fails and falls back to main channel",
			setup: func() {
				mockSession.EXPECT().GetChannel("channel-123").Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage("channel-123", "12345").
					Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive("channel-123").
					Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().
					MessageThreadStartComplex("channel-123", "12345", gomock.Any()).
					Return(nil, errors.New("thread create failed"))
				mockSession.EXPECT().
					ChannelMessageSend("channel-123", gomock.Any()).
					Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
		},
		{
			name: "send to thread fails",
			setup: func() {
				mockSession.EXPECT().GetChannel("channel-123").Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage("channel-123", "12345").
					Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive("channel-123").
					Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().
					MessageThreadStartComplex("channel-123", "12345", gomock.Any()).
					Return(&discordgo.Channel{ID: "thread-123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend("thread-123", gomock.Any()).
					Return(nil, errors.New("send failed"))
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to send reminder message")},
			wantErr: true,
		},
		{
			name: "thread already exists via race condition",
			setup: func() {
				mockSession.EXPECT().GetChannel("channel-123").Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage("channel-123", "12345").
					Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive("channel-123").
					Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().
					MessageThreadStartComplex("channel-123", "12345", gomock.Any()).
					Return(nil, errors.New("Thread already exists"))
				mockSession.EXPECT().
					ChannelMessage("channel-123", "12345").
					Return(&discordgo.Message{
						Thread: &discordgo.Channel{ID: "thread-123"},
					}, nil)
				mockSession.EXPECT().
					ChannelMessageSend("thread-123", gomock.Any()).
					Return(&discordgo.Message{}, nil)
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
				session: mockSession,
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
