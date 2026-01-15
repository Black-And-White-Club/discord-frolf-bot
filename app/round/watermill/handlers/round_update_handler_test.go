package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundUpdateRequested(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round Update")
	testDescription := roundtypes.Description("Updated Description")

	tests := []struct {
		name       string
		payload    *discordroundevents.RoundUpdateModalSubmittedPayloadV1
		ctx        context.Context
		want       []handlerwrapper.Result
		wantErr    bool
		wantLen    int // Expected number of results
	}{
		{
			name: "successful_update_request",
			payload: &discordroundevents.RoundUpdateModalSubmittedPayloadV1{
				GuildID:     "123456789",
				RoundID:     testRoundID,
				Title:       &testTitle,
				Description: &testDescription,
				UserID:      "user123",
				ChannelID:   "channel123",
				MessageID:   "message123",
			},
			ctx:     context.Background(),
			want:    nil, // We'll check the length instead of deep equality
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "create_result_message_error",
			payload: &discordroundevents.RoundUpdateModalSubmittedPayloadV1{
				GuildID:   "123456789",
				RoundID:   testRoundID,
				Title:     &testTitle,
				UserID:    "user123",
				ChannelID: "channel123",
				MessageID: "message123",
			},
			ctx:     context.Background(),
			want:    nil,
			wantErr: false,
			wantLen: 1,
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

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundUpdateRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundUpdateRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundUpdateRequestedV1 {
					t.Errorf("HandleRoundUpdateRequested() topic = %s, want %s", result.Topic, roundevents.RoundUpdateRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundUpdateRequested() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "test-channel-id"
	testMessageID := "test-message-id"
	testTitle := roundtypes.Title("Updated Round Title")
	testDescription := roundtypes.Description("Updated Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.LocationPtr("Updated Location")

	tests := []struct {
		name       string
		payload    *roundevents.RoundEntityUpdatedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int // Expected number of results
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockUpdateRoundManager)
	}{
		{
			name: "successful_round_updated",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:          testRoundID,
					Title:       testTitle,
					Description: &testDescription,
					StartTime:   &testStartTime,
					Location:    testLocation,
				},
			},
			ctx: context.WithValue(context.WithValue(context.Background(), "channel_id", testChannelID), "message_id", testMessageID),
			wantErr: false,
			wantLen: 0, // Side-effect only
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testMessageID,
					&testTitle,
					&testDescription,
					&testStartTime,
					testLocation,
				).Return(updateround.UpdateRoundOperationResult{}, nil).Times(1)
			},
		},
		{
			name: "update_with_nil_fields",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:       testRoundID,
					Title:    testTitle,
					Location: testLocation,
				},
			},
			ctx: context.WithValue(context.WithValue(context.Background(), "channel_id", testChannelID), "message_id", testMessageID),
			wantErr: false,
			wantLen: 0, // Side-effect only
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testMessageID,
					&testTitle,
					nil,
					nil,
					testLocation,
				).Return(updateround.UpdateRoundOperationResult{}, nil).Times(1)
			},
		},
		{
			name: "update_embed_error",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:    testRoundID,
					Title: testTitle,
				},
			},
			ctx: context.WithValue(context.WithValue(context.Background(), "channel_id", testChannelID), "message_id", testMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testMessageID,
					&testTitle,
					nil,
					nil,
					nil,
				).Return(updateround.UpdateRoundOperationResult{
					Error: errors.New("failed to update embed"),
				}, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockUpdateRoundManager := mocks.NewMockUpdateRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockUpdateRoundManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundUpdated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundUpdated() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateFailed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testError := "Round update failed due to database error"
	testTitle := roundtypes.Title("Test Title")
	testDescription := roundtypes.Description("Test Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.Location("Test Location")
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name       string
		payload    *roundevents.RoundUpdateErrorPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int // Expected number of results
	}{
		{
			name: "handle_update_failed_with_full_payload",
			payload: &roundevents.RoundUpdateErrorPayloadV1{
				RoundUpdateRequest: &roundevents.RoundUpdateRequestPayloadV1{
					RoundID:     testRoundID,
					Title:       testTitle,
					Description: &testDescription,
					StartTime:   &testStartTime,
					Location:    &testLocation,
					UserID:      testUserID,
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0, // Side-effect only
		},
		{
			name: "handle_update_failed_minimal_payload",
			payload: &roundevents.RoundUpdateErrorPayloadV1{
				RoundUpdateRequest: &roundevents.RoundUpdateRequestPayloadV1{
					RoundID: testRoundID,
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0, // Side-effect only
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

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundUpdateFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundUpdateFailed() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateValidationFailed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("Test Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.Location("Test Location")
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name       string
		payload    *roundevents.RoundUpdateValidatedPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int // Expected number of results
	}{
		{
			name: "handle_validation_failed_full_payload",
			payload: &roundevents.RoundUpdateValidatedPayloadV1{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					RoundID:     testRoundID,
					Title:       testTitle,
					Description: &testDescription,
					StartTime:   &testStartTime,
					Location:    &testLocation,
					UserID:      testUserID,
				},
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0, // Side-effect only
		},
		{
			name: "handle_validation_failed_minimal_payload",
			payload: &roundevents.RoundUpdateValidatedPayloadV1{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayloadV1{
					RoundID: testRoundID,
				},
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0, // Side-effect only
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

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
			}

			got, err := h.HandleRoundUpdateValidationFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundUpdateValidationFailed() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}
