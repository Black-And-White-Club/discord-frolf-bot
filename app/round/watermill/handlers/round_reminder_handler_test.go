package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	roundreminder "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_reminder"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testReminderType := "start"

	tests := []struct {
		name       string
		payload    *roundevents.DiscordReminderPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockRoundReminderManager)
	}{
		{
			name: "successful_reminder_sent",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:      testRoundID,
				ReminderType: testReminderType,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(roundreminder.RoundReminderOperationResult{
					Success: true,
				}, nil).Times(1)
			},
		},
		{
			name: "reminder_send_error",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:      testRoundID,
				ReminderType: testReminderType,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundreminder.RoundReminderOperationResult{},
					errors.New("send error"),
				).Times(1)
			},
		},
		{
			name: "operation_failure",
			payload: &roundevents.DiscordReminderPayloadV1{
				RoundID:      testRoundID,
				ReminderType: testReminderType,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundreminder.RoundReminderOperationResult{
						Success: false,
					},
					nil,
				).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockReminderManager := mocks.NewMockRoundReminderManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			tt.setup(ctrl, mockRoundDiscord, mockReminderManager)

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{
					Discord: config.DiscordConfig{
						EventChannelID: "test-channel-id",
					},
				},
				nil,
				mockRoundDiscord,
				nil,
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
