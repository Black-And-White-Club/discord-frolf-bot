package roundhandlers

import (
	"testing"

	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.uber.org/mock/gomock"
)

func TestNewRoundHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRoundDiscord := rounddiscord.NewMockRoundDiscordInterface(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		cfg := &config.Config{}

		handlers := NewRoundHandlers(logger, cfg, mockHelpers, mockRoundDiscord, nil)

		if handlers == nil {
			t.Fatalf("Expected non-nil RoundHandlers")
		}
	})
}
