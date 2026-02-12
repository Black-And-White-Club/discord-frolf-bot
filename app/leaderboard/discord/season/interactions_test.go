package season

import (
	"context"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel"
)

func TestSeasonManager_HandleSeasonCommand_Start(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	fakeHelpers := &testutils.FakeHelpers{}
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	logger := testutils.NoOpLogger()
	cfg := &config.Config{}
	
	// Create manager
	manager := NewSeasonManager(
		fakeSession,
		fakeEventBus,
		logger,
		fakeHelpers,
		cfg,
		nil, // GuildConfigResolver not needed for this test
		nil, // InteractionStore
		nil, // GuildConfigCache
		otel.Tracer("test"),
		fakeMetrics,
	)

	// Mock InteractionRespond (Deferred)
	respondCalled := false
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		respondCalled = true
		if resp.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
			t.Errorf("expected deferred response type, got %v", resp.Type)
		}
		return nil
	}

	// Mock EventBus Publish
	publishCalled := false
	fakeEventBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		publishCalled = true
		if topic != leaderboardevents.LeaderboardStartNewSeasonV1 {
			t.Errorf("expected topic %s, got %s", leaderboardevents.LeaderboardStartNewSeasonV1, topic)
		}
		return nil
	}

	// Mock Helper CreateNewMessage
	fakeHelpers.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
		return message.NewMessage("test-uuid", nil), nil
	}

	// Mock InteractionResponseEdit
	editCalled := false
	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		editCalled = true
		return &discordgo.Message{}, nil
	}

	// Create interaction
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			ID:      "interaction-id",
			GuildID: "guild-id",
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user-id", Username: "User"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "start",
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "name",
								Value: "Spring 2026",
								Type:  discordgo.ApplicationCommandOptionString,
							},
						},
					},
				},
			},
		},
	}

	// Execute
	manager.HandleSeasonCommand(context.Background(), interaction)

	// Assert
	if !respondCalled {
		t.Error("expected InteractionRespond to be called")
	}
	if !publishCalled {
		t.Error("expected EventBus.Publish to be called")
	}
	if !editCalled {
		t.Error("expected InteractionResponseEdit to be called")
	}
}

func TestSeasonManager_HandleSeasonCommand_Standings(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeEventBus := &testutils.FakeEventBus{}
	fakeHelpers := &testutils.FakeHelpers{}
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	logger := testutils.NoOpLogger()
	cfg := &config.Config{}

	manager := NewSeasonManager(
		fakeSession,
		fakeEventBus,
		logger,
		fakeHelpers,
		cfg,
		nil,
		nil,
		nil,
		otel.Tracer("test"),
		fakeMetrics,
	)

	// Mock InteractionRespond
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		return nil
	}

	// Mock EventBus Publish
	publishCalled := false
	fakeEventBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		publishCalled = true
		if topic != leaderboardevents.LeaderboardGetSeasonStandingsV1 {
			t.Errorf("expected topic %s, got %s", leaderboardevents.LeaderboardGetSeasonStandingsV1, topic)
		}
		return nil
	}

	fakeHelpers.CreateNewMessageFunc = func(payload interface{}, topic string) (*message.Message, error) {
		return message.NewMessage("test-uuid", nil), nil
	}

	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{}, nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			ID:      "interaction-id",
			GuildID: "guild-id",
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user-id", Username: "User"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "standings",
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "season_id",
								Value: "00000000-0000-0000-0000-000000000000",
								Type:  discordgo.ApplicationCommandOptionString,
							},
						},
					},
				},
			},
		},
	}

	manager.HandleSeasonCommand(context.Background(), interaction)

	if !publishCalled {
		t.Error("expected EventBus.Publish to be called")
	}
}

func TestSeasonManager_HandleSeasonCommand_Standings_InvalidUUID(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	logger := testutils.NoOpLogger()
	fakeMetrics := &testutils.FakeDiscordMetrics{}
	
	manager := NewSeasonManager(
		fakeSession,
		nil,
		logger,
		nil,
		nil,
		nil,
		nil,
		nil,
		otel.Tracer("test"),
		fakeMetrics,
	)

	errorResponded := false
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		if resp.Data.Content == "Error: Invalid season_id provided. Must be a valid UUID." {
			errorResponded = true
		}
		return nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:    discordgo.InteractionApplicationCommand,
			ID:      "interaction-id",
			GuildID: "guild-id",
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user-id", Username: "User"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "standings",
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "season_id",
								Value: "invalid-uuid",
								Type:  discordgo.ApplicationCommandOptionString,
							},
						},
					},
				},
			},
		},
	}

	manager.HandleSeasonCommand(context.Background(), interaction)

	if !errorResponded {
		t.Error("expected error response for invalid UUID")
	}
}
