package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundUpdateRequested(t *testing.T) {
	testTitle := roundtypes.Title("Updated Round")
	testDesc := roundtypes.Description("Updated Description")
	testLoc := roundtypes.Location("Updated Location")
	testTimeStr := "2024-01-01T12:00:00Z"
	testTimezone := roundtypes.Timezone("America/New_York")

	tests := []struct {
		name    string
		payload *discordroundevents.RoundUpdateModalSubmittedPayloadV1
		ctx     context.Context
		want    []handlerwrapper.Result
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_round_update_request",
			payload: &discordroundevents.RoundUpdateModalSubmittedPayloadV1{
				RoundID:     sharedtypes.RoundID(uuid.New()),
				Title:       &testTitle,
				Description: &testDesc,
				Location:    &testLoc,
				StartTime:   &testTimeStr,
				Timezone:    &testTimezone,
			},
			ctx:     context.Background(),
			want:    nil, // Checked via length
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
			)

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
	testTitle := roundtypes.Title("Updated Round")
	testDescription := roundtypes.Description("Updated Description")
	testLocation := roundtypes.Location("Updated Location")
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedTime)
	testChannelID := "channel-123"
	testMessageID := "message-123"

	tests := []struct {
		name    string
		payload *roundevents.RoundEntityUpdatedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_update",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:          testRoundID,
					Title:       testTitle,
					Description: testDescription,
					Location:    testLocation,
					StartTime:   &testStartTime,
				},
			},
			ctx:     context.WithValue(context.WithValue(context.Background(), "channel_id", testChannelID), "message_id", testMessageID),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.UpdateRoundManager.UpdateRoundEventEmbedFunc = func(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (updateround.UpdateRoundOperationResult, error) {
					return updateround.UpdateRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "embed_update_fails",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID:    testRoundID,
					Title: testTitle,
				},
			},
			ctx:     context.WithValue(context.WithValue(context.Background(), "channel_id", testChannelID), "message_id", testMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.UpdateRoundManager.UpdateRoundEventEmbedFunc = func(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (updateround.UpdateRoundOperationResult, error) {
					return updateround.UpdateRoundOperationResult{}, errors.New("failed to update embed")
				}
			},
		},
		{
			name: "missing_channel_id",
			payload: &roundevents.RoundEntityUpdatedPayloadV1{
				Round: roundtypes.Round{
					ID: testRoundID,
				},
			},
			ctx:     context.WithValue(context.Background(), "message_id", testMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				// No setup needed
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
			)

			got, err := h.HandleRoundUpdated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundUpdated() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.RoundUpdateErrorPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_update_failed",
			payload: &roundevents.RoundUpdateErrorPayloadV1{
				Error: "Test Error",
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: false,
			setup: func(f *FakeRoundDiscord) {
				// Handler returns nil nil without doing anything, so no setup needed if manager is not called
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
			)

			got, err := h.HandleRoundUpdateFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Handler should always return nil results (side-effect only)
			if len(got) > 0 {
				t.Errorf("HandleRoundUpdateFailed() expected nil or empty results, got %v", got)
			}
		})
	}
}
