package leaderboardhandlers

import (
	"testing"

	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLeaderboardDiscord := leaderboarddiscord.NewMockLeaderboardDiscordInterface(ctrl)
		mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)
		mockHelpersFn := mockHelpers.NewMockHelpers(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		cfg := &config.Config{}

		handlers := NewLeaderboardHandlers(
			logger,
			cfg,
			mockHelpersFn,
			mockLeaderboardDiscord,
			mockGuildConfigResolver,
		)

		if handlers == nil {
			t.Error("expected non-nil handlers")
		}
	})
}
