package userhandlers

import (
	"testing"

	mockdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock dependencies
	mockUserDiscord := mockdiscord.NewMockUserDiscordInterface(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	helpers := mockHelpers.NewMockHelpers(ctrl)
	cfg := &config.Config{}

	// Call the function being tested
	handlers := NewUserHandlers(logger, cfg, helpers, mockUserDiscord, tracer, nil)

	// Ensure handlers are correctly created
	if handlers == nil {
		t.Fatalf("NewUserHandlers returned nil")
	}

	// Access userHandlers directly from the Handler interface
	userHandlers := handlers.(*UserHandlers)

	// Check that all dependencies were correctly assigned
	if userHandlers.userDiscord != mockUserDiscord {
		t.Errorf("userDiscord not correctly assigned")
	}
	if userHandlers.logger != logger {
		t.Errorf("logger not correctly assigned")
	}
	if userHandlers.tracer != tracer {
		t.Errorf("tracer not correctly assigned")
	}
	if userHandlers.helper != helpers {
		t.Errorf("helper not correctly assigned")
	}
	if userHandlers.config != cfg {
		t.Errorf("config not correctly assigned")
	}
}

func TestNewUserHandlersWithNilDependencies(t *testing.T) {
	// Call with nil dependencies
	handlers := NewUserHandlers(nil, nil, nil, nil, nil, nil)

	// Ensure handlers are correctly created
	if handlers == nil {
		t.Fatalf("NewUserHandlers returned nil")
	}

	// Check that it returns a Handler interface
	if _, ok := handlers.(Handler); !ok {
		t.Errorf("handlers does not implement Handler interface")
	}
}
