package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.uber.org/mock/gomock"
)

func TestGuildHandlers_HandleGuildConfigCreated(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigCreatedPayloadV1
		wantErr bool
		wantLen int
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *guildconfigmocks.MockGuildConfigResolver)
	}{
		{
			name: "successful guild config created",
			payload: &guildevents.GuildConfigCreatedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Config: guildtypes.GuildConfig{
					GuildID:              sharedtypes.GuildID("123456789"),
					SignupChannelID:      "signup-channel",
					EventChannelID:       "event-channel",
					LeaderboardChannelID: "leaderboard-channel",
				},
			},
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockGuildConfigResolver *guildconfigmocks.MockGuildConfigResolver) {
				mockGuildDiscord.EXPECT().RegisterAllCommands("123456789").Return(nil).Times(1)
				mockGuildConfigResolver.EXPECT().HandleGuildConfigReceived(gomock.Any(), "123456789", gomock.Any()).Times(1)
			},
		},
		{
			name: "failed to register commands",
			payload: &guildevents.GuildConfigCreatedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Config: guildtypes.GuildConfig{
					GuildID:              sharedtypes.GuildID("123456789"),
					SignupChannelID:      "signup-channel",
					EventChannelID:       "event-channel",
					LeaderboardChannelID: "leaderboard-channel",
				},
			},
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockGuildConfigResolver *guildconfigmocks.MockGuildConfigResolver) {
				mockGuildDiscord.EXPECT().RegisterAllCommands("123456789").Return(fmt.Errorf("failed to register")).Times(1)
			},
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGuildDiscord := mocks.NewMockGuildDiscordInterface(ctrl)
			mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

			if tt.setup != nil {
				tt.setup(ctrl, mockGuildDiscord, mockGuildConfigResolver)
			}

			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				&config.Config{},
				mockGuildDiscord,
				mockGuildConfigResolver,
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigCreated(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigCreationFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config creation failed",
			payload: &guildevents.GuildConfigCreationFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "database connection failed",
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				nil, // config
				nil, // guildDiscord
				nil, // guildConfigResolver
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigCreationFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}
