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
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandlePointsAwarded(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testMessageID := "msg-123"
	testChannelID := "channel-123"
	user1 := sharedtypes.DiscordID("user-1")
	user2 := sharedtypes.DiscordID("user-2")

	tests := []struct {
		name    string
		payload *sharedevents.PointsAwardedPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_points_awarded",
			payload: &sharedevents.PointsAwardedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Points: map[sharedtypes.DiscordID]int{
					user1: 10,
					user2: 5,
				},
			},
			ctx: context.WithValue(context.Background(), "discord_message_id", testMessageID),
			setup: func(f *FakeRoundDiscord) {
				f.FinalizeRoundManager.FinalizeScorecardEmbedFunc = func(ctx context.Context, msgID, chID string, payload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error) {
					if msgID != testMessageID {
						return finalizeround.FinalizeRoundOperationResult{}, errors.New("wrong msg id")
					}
					// Verify points updated
					for _, p := range payload.Participants {
						if p.UserID == user1 && (p.Points == nil || *p.Points != 10) {
							return finalizeround.FinalizeRoundOperationResult{}, errors.New("user1 points not updated")
						}
						if p.UserID == user2 && (p.Points == nil || *p.Points != 5) {
							return finalizeround.FinalizeRoundOperationResult{}, errors.New("user2 points not updated")
						}
					}
					return finalizeround.FinalizeRoundOperationResult{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "round_payload_not_in_cache",
			payload: &sharedevents.PointsAwardedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Points:  map[sharedtypes.DiscordID]int{user1: 10},
			},
			ctx: context.Background(),
			setup: func(f *FakeRoundDiscord) {

			},
			wantErr: false, // Should return nil error to avoid retries
		},
		{
			name: "missing_discord_message_id",
			payload: &sharedevents.PointsAwardedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Points:  map[sharedtypes.DiscordID]int{user1: 10},
			},
			ctx: context.Background(),
			setup: func(f *FakeRoundDiscord) {

			},
			wantErr: false,
		},
		{
			name: "finalize_embed_fails",
			payload: &sharedevents.PointsAwardedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Points:  map[sharedtypes.DiscordID]int{user1: 10},
			},
			ctx: context.WithValue(context.Background(), "discord_message_id", testMessageID),
			setup: func(f *FakeRoundDiscord) {
				f.FinalizeRoundManager.FinalizeScorecardEmbedFunc = func(ctx context.Context, msgID, chID string, payload roundevents.RoundFinalizedEmbedUpdatePayloadV1) (finalizeround.FinalizeRoundOperationResult, error) {
					return finalizeround.FinalizeRoundOperationResult{}, errors.New("api error")
				}
			},
			wantErr: true,
		},
		{
			name: "no_matching_participants",
			payload: &sharedevents.PointsAwardedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Points:  map[sharedtypes.DiscordID]int{sharedtypes.DiscordID("unknown"): 10},
			},
			ctx: context.WithValue(context.Background(), "discord_message_id", testMessageID),
			setup: func(f *FakeRoundDiscord) {
			},
			wantErr: false, // Handled as warning, returns nil results
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
						EventChannelID: testChannelID,
					},
				},
				nil,
				fakeRoundDiscord,
				nil,
			)

			_, err := h.HandlePointsAwarded(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandlePointsAwarded() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
