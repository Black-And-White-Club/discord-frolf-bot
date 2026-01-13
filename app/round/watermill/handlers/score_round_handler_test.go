package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "test-channel"
	testParticipant := sharedtypes.DiscordID("user123")
	testScore := sharedtypes.Score(+18)
	testEventMessageID := "event-msg-123"

	tests := []struct {
		name    string
		payload *roundevents.ParticipantScoreUpdatedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockScoreRoundManager)
	}{
		{
			name: "successful_score_update",
			payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				RoundID:        testRoundID,
				ChannelID:      testChannelID,
				UserID:         testParticipant,
				Score:          testScore,
				EventMessageID: testEventMessageID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreManager).
					AnyTimes()

				mockScoreManager.EXPECT().
					UpdateScoreEmbed(
						gomock.Any(),
						testChannelID,
						testEventMessageID,
						testParticipant,
						&testScore,
					).Return(scoreround.ScoreRoundOperationResult{}, nil).Times(1)
			},
		},
		{
			name: "update_score_embed_fails",
			payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				RoundID:        testRoundID,
				ChannelID:      testChannelID,
				UserID:         testParticipant,
				Score:          testScore,
				EventMessageID: testEventMessageID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreManager).
					AnyTimes()

				mockScoreManager.EXPECT().
					UpdateScoreEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(scoreround.ScoreRoundOperationResult{}, errors.New("update failed")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockScoreManager := mocks.NewMockScoreRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockScoreManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleParticipantScoreUpdated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleParticipantScoreUpdated() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateError(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("user123")
	testError := "database connection failed"
	scoreZero := sharedtypes.Score(0)

	tests := []struct {
		name    string
		payload *roundevents.RoundScoreUpdateErrorPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockScoreRoundManager)
	}{
		{
			name: "successful_error_handling",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     &scoreZero,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreManager).
					AnyTimes()

				mockScoreManager.EXPECT().
					SendScoreUpdateError(gomock.Any(), testParticipant, testError).
					Return(scoreround.ScoreRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name:    "invalid_payload_type",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				// No setup needed for invalid payload
			},
		},
		{
			name: "empty_error_message",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     &scoreZero,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
				Error: "",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				// No setup needed for empty error
			},
		},
		{
			name: "send_error_fails",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     &scoreZero,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockScoreManager *mocks.MockScoreRoundManager) {
				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreManager).
					AnyTimes()

				mockScoreManager.EXPECT().
					SendScoreUpdateError(gomock.Any(), testParticipant, testError).
					Return(scoreround.ScoreRoundOperationResult{}, errors.New("send failed")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockScoreManager := mocks.NewMockScoreRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockScoreManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleScoreUpdateError(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScoreUpdateError() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundTraceEventV1 {
					t.Errorf("HandleScoreUpdateError() topic = %s, want %s", result.Topic, roundevents.RoundTraceEventV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleScoreUpdateError() payload is nil")
				}
			}
		})
	}
}
