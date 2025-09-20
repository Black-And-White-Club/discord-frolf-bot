package user

import (
	"context"
	"log/slog"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestInitializeUserModule_Succeeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	session := discordmocks.NewMockSession(ctrl)
	publisher := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(loggerfrolfbot.NewTestHandler())
	helper := utilsmocks.NewMockHelpers(ctrl)
	cfg := &config.Config{}
	interactionStore := &mockInteractionStore{}
	metrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	guildCfg := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	ireg := interactions.NewRegistry()
	rreg := interactions.NewReactionRegistry()

	userRouter, initErr := InitializeUserModule(ctx, session, router, ireg, rreg, publisher, logger, cfg, helper, interactionStore, metrics, guildCfg)
	if initErr != nil {
		t.Fatalf("InitializeUserModule returned error: %v", initErr)
	}
	if userRouter == nil || userRouter.Router == nil {
		t.Fatalf("expected non-nil user router")
	}

	// Note: We don't call Close() on the router here because the Watermill
	// router wasn't started in this unit test, and calling Close() without
	// Run() can lead to a timeout error from the library.
}

// minimal mock for storage.ISInterface to avoid pulling concrete store with timers
type mockInteractionStore struct{}

func (m *mockInteractionStore) Set(string, interface{}, time.Duration) error { return nil }
func (m *mockInteractionStore) Delete(string)                                {}
func (m *mockInteractionStore) Get(string) (interface{}, bool)               { return nil, false }
