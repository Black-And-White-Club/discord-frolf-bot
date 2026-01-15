package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
)

func TestHandleBatchTagAssigned(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1
		wantErr bool
		setup   func(*gomock.Controller) (*leaderboarddiscord.MockLeaderboardDiscordInterface, *guildconfigmocks.MockGuildConfigResolver)
	}{
		{
			name: "empty_assignments",
			payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				BatchID:     "batch1",
				Assignments: []leaderboardevents.TagAssignmentInfoV1{},
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller) (*leaderboarddiscord.MockLeaderboardDiscordInterface, *guildconfigmocks.MockGuildConfigResolver) {
				// No guild config resolver to avoid unexpected calls during this test
				return leaderboarddiscord.NewMockLeaderboardDiscordInterface(ctrl), nil
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
			setup: func(ctrl *gomock.Controller) (*leaderboarddiscord.MockLeaderboardDiscordInterface, *guildconfigmocks.MockGuildConfigResolver) {
				mockDiscord := leaderboarddiscord.NewMockLeaderboardDiscordInterface(ctrl)
				// Not providing a guild config resolver to keep behavior deterministic in tests
				return mockDiscord, nil
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	cfg := &config.Config{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDiscord, mockGuildConfig := tt.setup(ctrl)

			// Ensure we don't assign a typed nil to the interface field.
			var guildResolver guildconfig.GuildConfigResolver
			if mockGuildConfig != nil {
				guildResolver = mockGuildConfig
			}

			h := &LeaderboardHandlers{
				Logger:              logger,
				Tracer:              tracer,
				Config:              cfg,
				LeaderboardDiscord:  mockDiscord,
				GuildConfigResolver: guildResolver,
			}

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
