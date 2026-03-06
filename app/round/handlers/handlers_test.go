package handlers

import (
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
)

func TestNewRoundHandlers(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Run("Constructs handler with dependencies", func(t *testing.T) {
				logger := slog.Default()
				cfg := &config.Config{}
				fakeRoundDiscord := &FakeRoundDiscord{}

				handlers := NewRoundHandlers(logger, cfg, nil, fakeRoundDiscord, nil)

				if handlers == nil {
					t.Fatalf("Expected non-nil RoundHandlers")
				}
			})
		})
	}
}
