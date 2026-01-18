package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestHandleLeaderboardUpdateFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardUpdateFailedPayloadV1
		wantErr bool
	}{
		{
			name: "leaderboard_update_failed",
			payload: &leaderboardevents.LeaderboardUpdateFailedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				Reason:  "database connection timeout",
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := NewLeaderboardHandlers(
		logger,
		nil,
		nil,
		nil,
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleLeaderboardUpdateFailed(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// This handler returns empty results (just logs)
			if !tt.wantErr && len(results) != 0 {
				t.Errorf("expected empty results, got %d", len(results))
			}
		})
	}
}

func TestHandleLeaderboardRetrievalFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.GetLeaderboardFailedPayloadV1
		wantErr bool
	}{
		{
			name: "leaderboard_retrieval_failed",
			payload: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				Reason:  "leaderboard not found",
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := NewLeaderboardHandlers(
		logger,
		nil,
		nil,
		nil,
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleLeaderboardRetrievalFailed(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardRetrievalFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// This handler returns empty results (just logs)
			if !tt.wantErr && len(results) != 0 {
				t.Errorf("expected empty results, got %d", len(results))
			}
		})
	}
}
