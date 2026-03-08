package handlers

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
)

func TestNewUserHandlers(t *testing.T) {
	// Create fakes & mock dependencies
	fakeUserDiscord := &FakeUserDiscord{}
	logger := loggerfrolfbot.NoOpLogger
	helpers := &testutils.FakeHelpers{}
	cfg := &config.Config{}
	fakeGuildConfig := &FakeGuildConfigResolver{}

	// Call the function being tested
	handlers := NewUserHandlers(logger, cfg, helpers, fakeUserDiscord, fakeGuildConfig)

	// Verify returns non-nil handlers
	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
}

func TestNewUserHandlersWithNilDependencies(t *testing.T) {
	// Call with nil dependencies
	handlers := NewUserHandlers(nil, nil, nil, nil, nil)

	// Verify returns non-nil handlers
	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
}
