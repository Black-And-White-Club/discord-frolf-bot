package handlers

import (
	"context"
	"sync"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func TestGuildHandlers_CorrelationIDIsolationForConcurrentSetupResponses(t *testing.T) {
	session := discord.NewFakeSession()

	var (
		mu       sync.Mutex
		editedID = make(map[string]int)
	)
	session.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, _ *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
		mu.Lock()
		defer mu.Unlock()
		editedID[interaction.ID]++
		return &discordgo.Message{ID: "ok"}, nil
	}

	store := testutils.NewFakeStorage[any]()
	if err := store.Set(context.Background(), "corr-1", &discordgo.Interaction{ID: "interaction-1", GuildID: "g1"}); err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	if err := store.Set(context.Background(), "corr-2", &discordgo.Interaction{ID: "interaction-2", GuildID: "g1"}); err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}

	fakeGuildDiscord := &FakeGuildDiscord{
		RegisterAllCommandsFunc: func(guildID string) error { return nil },
	}

	handler := NewGuildHandlers(
		loggerfrolfbot.NoOpLogger,
		&config.Config{},
		fakeGuildDiscord,
		&guildconfig.FakeGuildConfigResolver{},
		nil,
		store,
		session,
	)

	payload := &guildevents.GuildConfigCreatedPayloadV1{
		GuildID: sharedtypes.GuildID("g1"),
		Config: guildtypes.GuildConfig{
			GuildID:              sharedtypes.GuildID("g1"),
			SignupChannelID:      "signup",
			EventChannelID:       "events",
			LeaderboardChannelID: "leaderboard",
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = handler.HandleGuildConfigCreated(context.WithValue(context.Background(), "correlation_id", "corr-1"), payload)
	}()
	go func() {
		defer wg.Done()
		_, _ = handler.HandleGuildConfigCreated(context.WithValue(context.Background(), "correlation_id", "corr-2"), payload)
	}()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if editedID["interaction-1"] != 1 || editedID["interaction-2"] != 1 {
		t.Fatalf("expected each interaction to be edited once, got %#v", editedID)
	}
}

func TestGuildHandlers_InteractionLookupFallbacksToGuildKey(t *testing.T) {
	session := discord.NewFakeSession()

	editedInteractionID := ""
	session.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, _ *discordgo.WebhookEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
		editedInteractionID = interaction.ID
		return &discordgo.Message{ID: "ok"}, nil
	}

	store := testutils.NewFakeStorage[any]()
	if err := store.Set(context.Background(), "g1", &discordgo.Interaction{ID: "legacy-interaction", GuildID: "g1"}); err != nil {
		t.Fatalf("failed to seed legacy key: %v", err)
	}

	handler := NewGuildHandlers(
		loggerfrolfbot.NoOpLogger,
		&config.Config{},
		&FakeGuildDiscord{},
		nil,
		nil,
		store,
		session,
	)

	_, err := handler.HandleGuildConfigCreationFailed(
		context.WithValue(context.Background(), "correlation_id", "missing-correlation"),
		&guildevents.GuildConfigCreationFailedPayloadV1{
			GuildID: sharedtypes.GuildID("g1"),
			Reason:  "forced failure",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if editedInteractionID != "legacy-interaction" {
		t.Fatalf("expected legacy interaction to be used, got %q", editedInteractionID)
	}
}
