package handlers

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/season"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/stretchr/testify/assert"
)

func TestHandleSeasonEndedResponse(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")

	t.Run("Calls SeasonManager", func(t *testing.T) {
		seasonMgr := &FakeSeasonManager{
			HandleSeasonEndedFunc: func(ctx context.Context, payload *leaderboardevents.EndSeasonSuccessPayloadV1) {
				assert.Equal(t, guildID, payload.GuildID)
			},
		}

		discordMock := &FakeLeaderboardDiscord{
			GetSeasonManagerFunc: func() season.SeasonManager {
				return seasonMgr
			},
		}

		handlers := &LeaderboardHandlers{
			service: discordMock,
			logger:  slog.Default(),
		}
		payload := &leaderboardevents.EndSeasonSuccessPayloadV1{GuildID: guildID}

		_, err := handlers.HandleSeasonEndedResponse(ctx, payload)
		assert.NoError(t, err)
	})
}

func TestHandleSeasonEndFailedResponse(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-123")
	reason := "some error"

	t.Run("Calls SeasonManager", func(t *testing.T) {
		seasonMgr := &FakeSeasonManager{
			HandleSeasonEndFailedFunc: func(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) {
				assert.Equal(t, guildID, payload.GuildID)
				assert.Equal(t, reason, payload.Reason)
			},
		}

		discordMock := &FakeLeaderboardDiscord{
			GetSeasonManagerFunc: func() season.SeasonManager {
				return seasonMgr
			},
		}

		handlers := &LeaderboardHandlers{
			service: discordMock,
			logger:  slog.Default(),
		}
		payload := &leaderboardevents.AdminFailedPayloadV1{GuildID: guildID, Reason: reason}

		_, err := handlers.HandleSeasonEndFailedResponse(ctx, payload)
		assert.NoError(t, err)
	})
}
