package userhandlers

import (
	"testing"

	mockdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	mockHelpers "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock dependencies
	mockUserDiscord := mockdiscord.NewMockUserDiscordInterface(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	helpers := mockHelpers.NewMockHelpers(ctrl)
	cfg := &config.Config{}

	// Call the function being tested
	handlers := NewUserHandlers(logger, cfg, helpers, mockUserDiscord)

	// Verify returns non-nil handlers
	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
}

func TestNewUserHandlersWithNilDependencies(t *testing.T) {
	// Call with nil dependencies
	handlers := NewUserHandlers(nil, nil, nil, nil)

	// Verify returns non-nil handlers
	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
}
