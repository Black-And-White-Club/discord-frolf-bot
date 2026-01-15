package scorehandlers

import (
	"testing"

	mockdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSession := mockdiscord.NewMockSession(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewScoreHandlers(logger, cfg, mockSession, mockHelpers, nil, nil, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil ScoreHandlers")
		}

		sh := handlers

		if sh.Session != mockSession {
			t.Errorf("Session not set correctly")
		}
		if sh.Helper != mockHelpers {
			t.Errorf("Helpers not set correctly")
		}
		if sh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if sh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if sh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if sh.Tracer != tracer {
			t.Errorf("Tracer not set correctly")
		}
	})
}

// Note: wrapHandler and message.HandlerFunc-style handlers were removed as part
// of the refactor to typed handlers. Tests for those wrappers are no longer
// applicable; behavior is covered by the typed handler unit tests in
// `score_handler_test.go`.
