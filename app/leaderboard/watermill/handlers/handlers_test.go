package leaderboardhandlers

import (
	"testing"

	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLeaderboardDiscord := leaderboarddiscord.NewMockLeaderboardDiscordInterface(ctrl)
		mockGuildConfigResolver := guildconfigmocks.NewMockGuildConfigResolver(ctrl)
		mockHelpersFn := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewLeaderboardHandlers(logger, cfg, mockHelpersFn, mockLeaderboardDiscord, mockGuildConfigResolver, nil, nil, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil LeaderboardHandlers")
		}

		lh := handlers.(*LeaderboardHandlers)

		if lh.LeaderboardDiscord != mockLeaderboardDiscord {
			t.Errorf("LeaderboardDiscord not set correctly")
		}
		if lh.GuildConfigResolver != mockGuildConfigResolver {
			t.Errorf("GuildConfigResolver not set correctly")
		}
		if lh.Helpers != mockHelpersFn {
			t.Errorf("Helpers not set correctly")
		}
		if lh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if lh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if lh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if lh.Tracer != tracer {
			t.Errorf("Tracer not set correctly")
		}
	})
}
