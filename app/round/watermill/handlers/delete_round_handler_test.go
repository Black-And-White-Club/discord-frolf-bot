package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundDeleteRequested(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name       string
		payload    *discordroundevents.RoundDeleteRequestDiscordPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface)
	}{
		{
			name: "successful_delete_request",
			payload: &discordroundevents.RoundDeleteRequestDiscordPayloadV1{
				RoundID: testRoundID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 1,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				// No additional setup needed for this handler
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundDeleteRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundDeleteRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundDeleteRequestedV1 {
					t.Errorf("HandleRoundDeleteRequested() topic = %s, want %s", result.Topic, roundevents.RoundDeleteRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundDeleteRequested() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testDiscordMessageID := "123456789012345678"

	tests := []struct {
		name       string
		payload    *roundevents.RoundDeletedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockDeleteRoundManager)
	}{
		{
			name: "successful_round_deletion",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: "some_other_id",
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", testDiscordMessageID),
			wantErr: false,
			wantLen: 0, // Handler intentionally returns empty results on success to avoid trace publish retries
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: true}, nil).
					Times(1)
			},
		},
		{
			name: "delete_round_event_embed_fails",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: "some_other_id",
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", testDiscordMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{}, errors.New("failed to delete round event embed")).
					Times(1)
			},
		},
		{
			name: "missing_discord_message_id_in_context",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: "some_other_id",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				// No mock setup needed since handler returns early when discord_message_id is missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockDeleteRoundManager := mocks.NewMockDeleteRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockDeleteRoundManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundDeleted(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundDeleted() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundTraceEventV1 {
					t.Errorf("HandleRoundDeleted() topic = %s, want %s", result.Topic, roundevents.RoundTraceEventV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundDeleted() payload is nil")
				}
			}
		})
	}
}
