package user

import (
	"context"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func TestInitializeUserModule_Succeeds(t *testing.T) {
	ctx := context.Background()
	session := &discord.FakeSession{}
	publisher := &testutils.FakeEventBus{}
	logger := slog.New(loggerfrolfbot.NewTestHandler())
	helper := &testutils.FakeHelpers{}
	cfg := &config.Config{}
	interactionStore := testutils.NewFakeStorage[any]()
	metrics := &discordmetrics.NoOpMetrics{}
	guildCfg := &testutils.FakeGuildConfigResolver{}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	ireg := interactions.NewRegistry()
	rreg := interactions.NewReactionRegistry(logger)

	userRouter, initErr := InitializeUserModule(ctx, session, router, ireg, rreg, publisher, logger, cfg, helper, interactionStore, nil, metrics, guildCfg)
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
