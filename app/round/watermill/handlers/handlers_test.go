package roundhandlers

import (
	"testing"

	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewRoundHandlers(t *testing.T) {
	t.Run("Constructs handler with dependencies", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRoundDiscord := rounddiscord.NewMockRoundDiscordInterface(ctrl)
		mockHelpers := mockHelpers.NewMockHelpers(ctrl)
		mockMetrics := mocks.NewMockDiscordMetrics(ctrl)
		logger := loggerfrolfbot.NoOpLogger
		tracer := noop.NewTracerProvider().Tracer("test")
		cfg := &config.Config{}

		handlers := NewRoundHandlers(logger, cfg, mockHelpers, mockRoundDiscord, tracer, mockMetrics)

		if handlers == nil {
			t.Fatalf("Expected non-nil RoundHandlers")
		}

		rh := handlers

		if rh.RoundDiscord != mockRoundDiscord {
			t.Errorf("RoundDiscord not set correctly")
		}
		if rh.Helpers != mockHelpers {
			t.Errorf("Helpers not set correctly")
		}
		if rh.Metrics != mockMetrics {
			t.Errorf("Metrics not set correctly")
		}
		if rh.Logger != logger {
			t.Errorf("Logger not set correctly")
		}
		if rh.Config != cfg {
			t.Errorf("Config not set correctly")
		}
		if rh.Tracer != tracer {
			t.Errorf("Tracer not set correctly")
		}
	})
}
