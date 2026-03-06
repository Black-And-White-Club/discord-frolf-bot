package scorehandlers

import (
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
)

func TestNewScoreHandlers(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			t.Run("Constructs handler with dependencies", func(t *testing.T) {
				logger := loggerfrolfbot.NoOpLogger

				handlers := NewScoreHandlers(logger)

				if handlers == nil {
					t.Fatalf("Expected non-nil Handlers")
				}
			})
		})
	}
}
