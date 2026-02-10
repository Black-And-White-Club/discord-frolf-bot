package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	roundreminder "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_reminder"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testReminderType := "start"

	tests := []struct {
		name    string
		payload *roundevents.DiscordReminderPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_reminder_sent",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:          testRoundID,
				ReminderType:     testReminderType,
				DiscordChannelID: "test-channel",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.RoundReminderManager.SendRoundReminderFunc = func(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (roundreminder.RoundReminderOperationResult, error) {
					return roundreminder.RoundReminderOperationResult{
						Success: true,
					}, nil
				}
			},
		},
		{
			name: "reminder_send_error",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:          testRoundID,
				ReminderType:     testReminderType,
				DiscordChannelID: "test-channel",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.RoundReminderManager.SendRoundReminderFunc = func(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (roundreminder.RoundReminderOperationResult, error) {
					return roundreminder.RoundReminderOperationResult{}, errors.New("failed to send reminder")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
				&FakeInteractionStore{},
			)

			got, err := h.HandleRoundReminder(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundReminder() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}
