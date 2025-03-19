package roundreminder

import (
	"context"
	"errors"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_roundReminderManager_SendRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	sampleTime := time.Now()
	sampleLocation := "Test Location"

	samplePayload := &roundevents.DiscordReminderPayload{
		RoundID:          123,
		RoundTitle:       "Test Round",
		UserIDs:          []roundtypes.UserID{"user-123", "user-456"},
		ReminderType:     "1-hour",
		DiscordChannelID: "channel-123",
		DiscordGuildID:   "guild-id",
		EventMessageID:   "message-123",
		StartTime:        (*roundtypes.StartTime)(&sampleTime),
		Location:         (*roundtypes.Location)(&sampleLocation),
	}

	type args struct {
		ctx     context.Context
		payload *roundevents.DiscordReminderPayload
	}

	tests := []struct {
		name    string
		setup   func()
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "successful reminder",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil).Times(1)
				mockSession.EXPECT().MessageThreadStartComplex(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).Return(&discordgo.Channel{ID: "thread-123"}, nil).Times(1)
				mockSession.EXPECT().ChannelMessageSend(gomock.Eq("thread-123"), gomock.Any()).Return(&discordgo.Message{}, nil).Times(1)
			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "failed to get channel",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(nil, errors.New("channel not found")).Times(1)
			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "failed to create thread",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil).Times(1)
				mockSession.EXPECT().MessageThreadStartComplex(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).Return(nil, errors.New("failed to create thread")).Times(1)
			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "failed to send message",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil).Times(1)
				mockSession.EXPECT().MessageThreadStartComplex(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).Return(&discordgo.Channel{ID: "thread-123"}, nil).Times(1)
				mockSession.EXPECT().ChannelMessageSend(gomock.Eq("thread-123"), gomock.Any()).Return(nil, errors.New("failed to send message")).Times(1)
			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "thread already exists",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil).Times(1)
				mockSession.EXPECT().MessageThreadStartComplex(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).Return(nil, errors.New("Thread already exists")).Times(1)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(
					&discordgo.ThreadsList{
						Threads: []*discordgo.Channel{
							{ID: "thread-123", ParentID: "channel-123", Name: "⏰ 1 Hour Reminder: Test Round"},
						},
					}, nil,
				).Times(1)

				mockSession.EXPECT().ChannelMessageSend(gomock.Eq("thread-123"), gomock.Any()).
					Return(&discordgo.Message{}, nil).Times(1)
			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "thread already exists, but cannot find it",
			setup: func() {
				mockSession.EXPECT().GetChannel(gomock.Eq("channel-123")).Return(&discordgo.Channel{}, nil).Times(1)
				mockSession.EXPECT().MessageThreadStartComplex(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).Return(nil, errors.New("Thread already exists")).Times(1)
				mockSession.EXPECT().ThreadsActive(gomock.Eq("guild-id")).Return(nil, nil).Times(1)

			},
			args: args{
				ctx:     context.Background(),
				payload: samplePayload,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "no message ID provided",
			setup: func() {
				samplePayloadNoMessageID := samplePayload
				samplePayloadNoMessageID.EventMessageID = ""
			},
			args: args{
				ctx: context.Background(),
				payload: &roundevents.DiscordReminderPayload{
					RoundID:          123,
					RoundTitle:       "Test Round",
					UserIDs:          []roundtypes.UserID{"user-123", "user-456"},
					ReminderType:     "1-hour",
					DiscordChannelID: "channel-123",
					DiscordGuildID:   "guild-id",
					EventMessageID:   "",
					StartTime:        (*roundtypes.StartTime)(&sampleTime),
					Location:         (*roundtypes.Location)(&sampleLocation),
				},
			},
			want:    false,
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
			}

			got, err := rm.SendRoundReminder(tt.args.ctx, tt.args.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("roundReminderManager.SendRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("roundReminderManager.SendRoundReminder() = %v, want %v", got, tt.want)
			}
		})
	}
}
