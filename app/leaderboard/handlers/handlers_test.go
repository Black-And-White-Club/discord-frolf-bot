package handlers

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
)

func TestNewLeaderboardHandlers(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Run("Constructs handler with dependencies", func(t *testing.T) {
				// Use Fakes instead of Mocks
				fakeLeaderboardDiscord := &FakeLeaderboardDiscord{}
				fakeGuildConfigResolver := &FakeGuildConfigResolver{}
				fakeHelpers := &FakeHelpers{}
				logger := loggerfrolfbot.NoOpLogger
				cfg := &config.Config{}

				handlers := NewLeaderboardHandlers(
					logger,
					cfg,
					fakeHelpers,
					fakeLeaderboardDiscord,
					fakeGuildConfigResolver,
				)

				if handlers == nil {
					t.Error("expected non-nil handlers")
				}
			})
		})
	}
}
