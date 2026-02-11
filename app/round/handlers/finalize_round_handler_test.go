package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	eventMessageID := sharedtypes.DiscordID("event-msg-123")
	configChannelID := "config-channel-123"

	tests := []struct {
		name    string
		payload *roundevents.RoundFinalizedDiscordPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_finalization",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:          testRoundID,
				DiscordChannelID: "payload-channel-123",
				EventMessageID:   string(eventMessageID),
				Participants:     []roundtypes.Participant{},
				Teams:            []roundtypes.NormalizedTeam{},
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", string(eventMessageID)),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.FinalizeRoundManager.FinalizeScorecardEmbedFunc = func(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error) {
					if channelID != "payload-channel-123" {
						return finalizeround.FinalizeRoundOperationResult{}, errors.New("wrong channel ID")
					}
					return finalizeround.FinalizeRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "finalize_scorecard_embed_fails",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: string(eventMessageID),
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", string(eventMessageID)),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.FinalizeRoundManager.FinalizeScorecardEmbedFunc = func(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error) {
					return finalizeround.FinalizeRoundOperationResult{}, errors.New("failed to finalize embed")
				}
			},
		},
		{
			name: "missing_discord_message_id_in_context",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:          testRoundID,
				DiscordChannelID: "1234",
			},
			ctx:     context.Background(), // Missing discord_message_id
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
				&config.Config{
					Discord: config.DiscordConfig{
						EventChannelID: configChannelID,
					},
				},
				nil,
				fakeRoundDiscord,
				&FakeGuildConfigResolver{},
			)

			got, err := h.HandleRoundFinalized(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != 0 {
				t.Errorf("HandleRoundFinalized() got %d results, want %d", len(got), 0)
				return
			}
		})
	}
}
