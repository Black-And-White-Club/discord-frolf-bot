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

	samplePayload := &roundevents.DiscordReminderPayload{
		RoundID:          sharedtypes.RoundID(testRoundID),
		RoundTitle:       "Test Round",
		UserIDs:          []sharedtypes.DiscordID{"user-123", "user-456"},
		ReminderType:     "1-hour",
		DiscordChannelID: "channel-123",
		DiscordGuildID:   "guild-id",
		EventMessageID:   "12345",
		StartTime:        (*sharedtypes.StartTime)(&sampleTime),
		Location:         (*roundtypes.Location)(&sampleLocation),
	}

	// Test case setup
	tests := []struct {
		name    string
		setup   func()
		payload *roundevents.DiscordReminderPayload
		want    RoundReminderOperationResult
		wantErr bool
	}{
		{
			name: "successful reminder",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(&discordgo.Channel{ID: "thread-123"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("thread-123"),
					gomock.Any(),
				).Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
			wantErr: false,
		},
		{
			name: "failed to get channel",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(nil, errors.New("channel not found"))
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to get channel: channel not found")},
			wantErr: true,
		},
		{
			name: "failed to create thread",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, errors.New("failed to create thread"))
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to create thread: failed to create thread")},
			wantErr: true,
		},
		{
			name: "failed to send message",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(&discordgo.Channel{ID: "thread-123"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("thread-123"),
					gomock.Any(),
				).Return(nil, errors.New("failed to send message"))
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Error: errors.New("failed to send reminder message to thread: failed to send message")},
			wantErr: true,
		},
		{
			name: "thread already exists",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, errors.New("Thread already exists"))
				// Additional ChannelMessage call in error handling path
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{
					ID:     "12345",
					Thread: &discordgo.Channel{ID: "thread-123", ParentID: "channel-123", Name: "⏰ 1 Hour Reminder: Test Round"},
				}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("thread-123"),
					gomock.Any(),
				).Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true},
			wantErr: false,
		},
		{
			name: "thread already exists, but cannot find it",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(&discordgo.ThreadsList{}, nil)
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, errors.New("Thread already exists"))
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(
					&discordgo.ThreadsList{
						Threads: []*discordgo.Channel{},
					}, nil)
				// Expect fallback to main channel when thread cannot be found
				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("channel-123"), // Main channel, not thread
					gomock.Any(),
				).Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true}, // Updated to expect success
			wantErr: false,                                       // Updated to expect no error
		},
		{
			name: "thread active fetch fails",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil)
				mockSession.EXPECT().ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("12345")).Return(&discordgo.Message{ID: "12345"}, nil)
				// When ThreadsActive fails, it continues and tries to create a new thread
				mockSession.EXPECT().MessageThreadStartComplex(
					gomock.Eq("channel-123"),
					gomock.Any(),
					gomock.Any(),
				).Return(&discordgo.Channel{ID: "thread-123"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					gomock.Eq("thread-123"),
					gomock.Any(),
				).Return(&discordgo.Message{}, nil)
			},
			payload: samplePayload,
			want:    RoundReminderOperationResult{Success: true}, // Should succeed by creating new thread
			wantErr: false,
		},
		{
			name: "no message ID provided",
			setup: func() {
				// No setup needed as we'll get early return
			},
			payload: &roundevents.DiscordReminderPayload{
				RoundID:          sharedtypes.RoundID(testRoundID),
				RoundTitle:       "Test Round",
				UserIDs:          []sharedtypes.DiscordID{"user-123", "user-456"},
				ReminderType:     "1-hour",
				DiscordChannelID: "channel-123",
				DiscordGuildID:   "guild-id",
				EventMessageID:   "", // This should be empty to trigger the error
				StartTime:        (*sharedtypes.StartTime)(&sampleTime),
				Location:         (*roundtypes.Location)(&sampleLocation),
			},
			want:    RoundReminderOperationResult{Error: errors.New("no message ID provided in payload")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			if tt.setup != nil {
				tt.setup()
			}

			// Create manager with mocks and bypass the operationWrapper
			rm := &roundReminderManager{
				session: mockSession,
				config:  mockConfig,
				logger:  mockLogger,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (RoundReminderOperationResult, error)) (RoundReminderOperationResult, error) {
					return fn(ctx)
				},
			}

			// Call the function
			got, err := rm.SendRoundReminder(context.Background(), tt.payload)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("roundReminderManager.SendRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For error messages, check if they match the expected error message
			if err != nil && tt.want.Error != nil {
				if !strings.Contains(err.Error(), tt.want.Error.Error()) {
					t.Errorf("Error message does not match expected. Got: %v, Want: %v", err, tt.want.Error)
				}
			}

			// Check the success field
			if got.Success != tt.want.Success {
				t.Errorf("roundReminderManager.SendRoundReminder() got = %v, want %v", got.Success, tt.want.Success)
			}
		})
	}
}
