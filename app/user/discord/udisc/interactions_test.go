package udisc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func udiscDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Test_udiscManager_HandleSetUDiscNameCommand_EmptyFields_EphemeralError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

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

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
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
		}).
		Times(1)

	m := &udiscManager{
		session:   mockSession,
		publisher: mockPublisher,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

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

	mockPublisher.EXPECT().
		Publish(gomock.Eq(userevents.UpdateUDiscIdentityRequest), gomock.Any()).
		DoAndReturn(func(_ string, msg *message.Message) error {
			var payload userevents.UpdateUDiscIdentityRequestPayload
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
		}).
		Times(1)

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
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
		}).
		Times(1)

	m := &udiscManager{
		session:   mockSession,
		publisher: mockPublisher,
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
