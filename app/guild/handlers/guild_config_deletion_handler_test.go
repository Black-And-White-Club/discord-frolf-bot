package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestGuildHandlers_HandleGuildConfigDeleted(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigDeletedPayloadV1
		wantErr bool
		wantLen int
		setup   func(*FakeGuildDiscord, *guildconfig.FakeGuildConfigResolver)
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
			setup: func(fakeGuildDiscord *FakeGuildDiscord, fakeGuildConfigResolver *guildconfig.FakeGuildConfigResolver) {
				fakeGuildDiscord.UnregisterAllCommandsFunc = func(guildID string) error {
					return nil
				}
				// Default FakeGuildDiscord.GetResetManager returns a fake ResetManager
				fakeGuildConfigResolver.ClearInflightRequestFunc = func(ctx context.Context, guildID string) {
					// Called
				}
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
			setup: func(fakeGuildDiscord *FakeGuildDiscord, fakeGuildConfigResolver *guildconfig.FakeGuildConfigResolver) {
				fakeGuildDiscord.UnregisterAllCommandsFunc = func(guildID string) error {
					return errors.New("failed to unregister")
				}
				fakeGuildConfigResolver.ClearInflightRequestFunc = func(ctx context.Context, guildID string) {
					// Called
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeGuildDiscord := &FakeGuildDiscord{}
			fakeGuildConfigResolver := &guildconfig.FakeGuildConfigResolver{}

			if tt.setup != nil {
				tt.setup(fakeGuildDiscord, fakeGuildConfigResolver)
			}

			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				&config.Config{},
				fakeGuildDiscord,
				fakeGuildConfigResolver,
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
