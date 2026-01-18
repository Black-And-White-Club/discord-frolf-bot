package handlers

import (
	"context"
	"errors"
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

func TestGuildHandlers_HandleGuildConfigDeleted(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigDeletedPayloadV1
		wantErr bool
		wantLen int
		setup   func(*gomock.Controller, *mocks.MockGuildDiscordInterface, *guildconfigmocks.MockGuildConfigResolver)
	}{
		{
			name: "successful guild config deleted",
			payload: &guildevents.GuildConfigDeletedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				ResourceState: guildtypes.ResourceState{
					SignupChannelID: "signup-channel",
				},
			},
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockGuildConfigResolver *guildconfigmocks.MockGuildConfigResolver) {
				mockGuildDiscord.EXPECT().UnregisterAllCommands("123456789").Return(nil).Times(1)
				// Handler may call GetResetManager; return nil to indicate no reset manager available
				mockGuildDiscord.EXPECT().GetResetManager().Return(nil).Times(1)
				mockGuildConfigResolver.EXPECT().ClearInflightRequest(gomock.Any(), "123456789").Times(1)
			},
		},
		{
			name: "failed to unregister commands",
			payload: &guildevents.GuildConfigDeletedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				ResourceState: guildtypes.ResourceState{
					SignupChannelID: "signup-channel",
				},
			},
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockGuildDiscord *mocks.MockGuildDiscordInterface, mockGuildConfigResolver *guildconfigmocks.MockGuildConfigResolver) {
				// Return a concrete error when unregistering commands
				mockGuildDiscord.EXPECT().UnregisterAllCommands("123456789").Return(errors.New("unregister error")).Times(1)
				mockGuildConfigResolver.EXPECT().ClearInflightRequest(gomock.Any(), "123456789").Times(1)
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

			results, err := h.HandleGuildConfigDeleted(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigDeletionFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigDeletionFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config deletion failed",
			payload: &guildevents.GuildConfigDeletionFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "backend error",
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

			results, err := h.HandleGuildConfigDeletionFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigDeletionFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}
