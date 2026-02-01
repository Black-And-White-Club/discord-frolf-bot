package handlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
)

func TestHandleBatchTagAssigned(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1
		wantErr bool
		setup   func() (*FakeLeaderboardDiscord, *FakeGuildConfigResolver)
	}{
		{
			name: "empty_assignments",
			payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				BatchID:     "batch1",
				Assignments: []leaderboardevents.TagAssignmentInfoV1{},
			},
			wantErr: false,
			setup: func() (*FakeLeaderboardDiscord, *FakeGuildConfigResolver) {
				// No guild config resolver to avoid unexpected calls during this test
				return &FakeLeaderboardDiscord{}, nil
			},
		},
		{
			name: "successful_batch_assigned",
			payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				BatchID: "batch1",
				Assignments: []leaderboardevents.TagAssignmentInfoV1{
					{
						TagNumber: sharedtypes.TagNumber(1),
						UserID:    sharedtypes.DiscordID("user1"),
					},
					{
						TagNumber: sharedtypes.TagNumber(2),
						UserID:    sharedtypes.DiscordID("user2"),
					},
				},
			},
			wantErr: false,
			setup: func() (*FakeLeaderboardDiscord, *FakeGuildConfigResolver) {
				fakeDiscord := &FakeLeaderboardDiscord{}
				// Not providing a guild config resolver to keep behavior deterministic in tests
				return fakeDiscord, nil
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDiscord, fakeGuildConfig := tt.setup()

			// Ensure we don't assign a typed nil to the interface field.
			var guildResolver guildconfig.GuildConfigResolver
			if fakeGuildConfig != nil {
				guildResolver = fakeGuildConfig
			}

			h := NewLeaderboardHandlers(
				logger,
				cfg,
				nil,
				fakeDiscord,
				guildResolver,
			)

			ctx := context.Background()
			results, err := h.HandleBatchTagAssigned(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBatchTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// If payload had assignments we expect results; if empty assignments, empty results are valid
				if len(tt.payload.Assignments) > 0 && len(results) == 0 {
					t.Errorf("expected results for non-empty assignments, got empty slice")
				}
			}
		})
	}
}
