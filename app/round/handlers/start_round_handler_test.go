package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	startround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/start_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundStarted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testEventMessageID := "event-message-123"
	testChannelID := "channel-123"
	discordMessageID := "discord-msg-123"

	tests := []struct {
		name    string
		payload *roundevents.DiscordRoundStartPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_started_transform_flow",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				EventMessageID:   testEventMessageID,
				DiscordChannelID: testChannelID,
				RoundID:          testRoundID,
				// Populate fields indicating transform flow if necessary,
				// but based on handler code it might branch based on something else or both use same payload
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.StartRoundManager.UpdateRoundToScorecardFunc = func(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (startround.StartRoundOperationResult, error) {
					return startround.StartRoundOperationResult{
						Success: &discordgo.Message{
							ID:        discordMessageID,
							ChannelID: testChannelID,
						},
					}, nil
				}
			},
		},
		{
			name: "missing_event_message_id",
			payload: &roundevents.DiscordRoundStartPayloadV1{
				// EventMessageID missing
				DiscordChannelID: testChannelID,
				RoundID:          testRoundID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
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
			setup: func(f *FakeRoundDiscord) {
				f.StartRoundManager.UpdateRoundToScorecardFunc = func(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (startround.StartRoundOperationResult, error) {
					return startround.StartRoundOperationResult{}, errors.New("update failed")
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
			)

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
