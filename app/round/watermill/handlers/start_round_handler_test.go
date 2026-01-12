package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	startround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/start_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundStarted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "channel123"
	testEventMessageID := "12345"

	tests := []struct {
		name       string
		payload    *roundevents.DiscordRoundStartPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockStartRoundManager)
	}{
		{
			name: "successful_round_start",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				EventMessageID:   testEventMessageID,
				DiscordChannelID: testChannelID,
				RoundID:          testRoundID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 1,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				mockRoundDiscord.EXPECT().
					GetStartRoundManager().
					Return(mockStartRoundManager).
					AnyTimes()

				mockStartRoundManager.EXPECT().
					UpdateRoundToScorecard(gomock.Any(), testChannelID, testEventMessageID, gomock.Any()).
					Return(startround.StartRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "invalid_payload_type",
			payload: &roundevents.DiscordRoundStartPayloadV1{},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				// No setup needed for invalid payload
			},
		},
		{
			name: "missing_event_message_id",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				DiscordChannelID: testChannelID,
				RoundID:          testRoundID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				// No setup needed for missing field
			},
		},
		{
			name: "update_round_to_scorecard_error",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				EventMessageID:   testEventMessageID,
				DiscordChannelID: testChannelID,
				RoundID:          testRoundID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				mockRoundDiscord.EXPECT().
					GetStartRoundManager().
					Return(mockStartRoundManager).
					AnyTimes()

				mockStartRoundManager.EXPECT().
					UpdateRoundToScorecard(gomock.Any(), testChannelID, testEventMessageID, gomock.Any()).
					Return(startround.StartRoundOperationResult{}, errors.New("update failed")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockStartRoundManager := mocks.NewMockStartRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockStartRoundManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundStarted(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStarted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundStarted() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundTraceEventV1 {
					t.Errorf("HandleRoundStarted() topic = %s, want %s", result.Topic, roundevents.RoundTraceEventV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundStarted() payload is nil")
				}
			}
		})
	}
}
