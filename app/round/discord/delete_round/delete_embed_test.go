package deleteround

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"go.uber.org/mock/gomock"
)

func Test_deleteRoundManager_DeleteEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	type args struct {
		ctx            context.Context
		eventMessageID roundtypes.EventMessageID
		channelID      string
	}

	tests := []struct {
		name    string
		setup   func()
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "successful delete",
			setup: func() {
				mockSession.EXPECT().ChannelMessageDelete(gomock.Eq("channel-123"), gomock.Eq("message-123")).Return(nil).Times(1)
			},
			args: args{
				ctx:            context.Background(),
				eventMessageID: "message-123",
				channelID:      "channel-123",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "missing channel ID",
			setup: func() {
				// No expectations needed since function should return early
			},
			args: args{
				ctx:            context.Background(),
				eventMessageID: "message-123",
				channelID:      "",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "missing event message ID",
			setup: func() {
				// No expectations needed since function should return early
			},
			args: args{
				ctx:            context.Background(),
				eventMessageID: "",
				channelID:      "channel-123",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "failed to delete message",
			setup: func() {
				mockSession.EXPECT().ChannelMessageDelete(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(errors.New("message not found")).Times(1)
			},
			args: args{
				ctx:            context.Background(),
				eventMessageID: "message-123",
				channelID:      "channel-123",
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

			dem := &deleteRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			got, err := dem.DeleteEmbed(tt.args.ctx, tt.args.eventMessageID, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteRoundManager.DeleteEmbed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("deleteRoundManager.DeleteEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}
