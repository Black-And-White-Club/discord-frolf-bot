package udisc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func udiscDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Test_udiscManager_HandleSetUDiscNameCommand_EmptyFields_EphemeralError(t *testing.T) {
	fakeSession := &discord.FakeSession{}
	fakePublisher := &testutils.FakeEventBus{}

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      "interaction-id",
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "guild-id",
		Member:  &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name:    "set-udisc-name",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{},
		},
	}}

	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		if interaction.ID != i.Interaction.ID {
			t.Errorf("expected interaction ID %q, got %q", i.Interaction.ID, interaction.ID)
		}
		if resp.Data == nil {
			t.Fatalf("expected response data")
		}
		if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
			t.Fatalf("expected ephemeral response")
		}
		if !strings.Contains(resp.Data.Content, "Please provide") {
			t.Fatalf("unexpected content: %q", resp.Data.Content)
		}
		return nil
	}

	m := &udiscManager{
		session:   fakeSession,
		publisher: fakePublisher,
		logger:    udiscDiscardLogger(),
		config:    &config.Config{Discord: config.DiscordConfig{GuildID: "guild-id"}},
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (UDiscOperationResult, error)) (UDiscOperationResult, error) {
			return fn(ctx)
		},
	}

	res, err := m.HandleSetUDiscNameCommand(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Error == nil {
		t.Fatalf("expected result error")
	}
}

func Test_udiscManager_HandleSetUDiscNameCommand_PublishesAndConfirms_UsesConfigGuildFallback(t *testing.T) {
	fakeSession := &discord.FakeSession{}
	fakePublisher := &testutils.FakeEventBus{}

	cfgGuildID := "cfg-guild"
	userID := "user-id"
	username := "@john  "
	name := " John Doe "

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      "interaction-id",
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "", // exercise config fallback
		Member:  &discordgo.Member{User: &discordgo.User{ID: userID}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: "set-udisc-name",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "username", Type: discordgo.ApplicationCommandOptionString, Value: username},
				{Name: "name", Type: discordgo.ApplicationCommandOptionString, Value: name},
			},
		},
	}}

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		if topic != userevents.UpdateUDiscIdentityRequestedV1 {
			t.Errorf("expected topic %q, got %q", userevents.UpdateUDiscIdentityRequestedV1, topic)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0]
		var payload userevents.UpdateUDiscIdentityRequestedPayloadV1
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if string(payload.GuildID) != cfgGuildID {
			t.Fatalf("guild_id mismatch: got %q want %q", payload.GuildID, cfgGuildID)
		}
		if string(payload.UserID) != userID {
			t.Fatalf("user_id mismatch: got %q want %q", payload.UserID, userID)
		}
		if payload.Username == nil || *payload.Username != strings.TrimSpace(username) {
			t.Fatalf("username mismatch: got %v", payload.Username)
		}
		if payload.Name == nil || *payload.Name != strings.TrimSpace(name) {
			t.Fatalf("name mismatch: got %v", payload.Name)
		}
		if msg.Metadata.Get("user_id") != userID {
			t.Fatalf("expected metadata user_id")
		}
		if msg.Metadata.Get("guild_id") != cfgGuildID {
			t.Fatalf("expected metadata guild_id")
		}
		if msg.Metadata.Get("correlation_id") == "" {
			t.Fatalf("expected metadata correlation_id")
		}
		return nil
	}

	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		if interaction.ID != i.Interaction.ID {
			t.Errorf("expected interaction ID %q, got %q", i.Interaction.ID, interaction.ID)
		}
		if resp.Data == nil {
			t.Fatalf("expected response data")
		}
		if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
			t.Fatalf("expected ephemeral response")
		}
		if !strings.Contains(resp.Data.Content, "UDisc identity updated") {
			t.Fatalf("unexpected content: %q", resp.Data.Content)
		}
		if !strings.Contains(resp.Data.Content, strings.TrimSpace(username)) {
			t.Fatalf("expected username in content: %q", resp.Data.Content)
		}
		if !strings.Contains(resp.Data.Content, strings.TrimSpace(name)) {
			t.Fatalf("expected name in content: %q", resp.Data.Content)
		}
		return nil
	}

	m := &udiscManager{
		session:   fakeSession,
		publisher: fakePublisher,
		logger:    udiscDiscardLogger(),
		config:    &config.Config{Discord: config.DiscordConfig{GuildID: cfgGuildID}},
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (UDiscOperationResult, error)) (UDiscOperationResult, error) {
			return fn(ctx)
		},
	}

	res, err := m.HandleSetUDiscNameCommand(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("expected nil result error, got %v", res.Error)
	}
	if res.Success != "udisc_name_set" {
		t.Fatalf("expected success %q, got %v", "udisc_name_set", res.Success)
	}
}
