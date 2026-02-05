package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestSendSetupModal_HappyPath(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	// Expect a modal response
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		if resp == nil || resp.Type != discordgo.InteractionResponseModal {
			t.Fatalf("expected modal response, got: %#v", resp)
		}
		return nil
	}

	m := &setupManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String(), Type: discordgo.InteractionApplicationCommand, GuildID: "g1"}}
	if err := m.SendSetupModal(context.Background(), i); err != nil {
		t.Fatalf("SendSetupModal returned error: %v", err)
	}
}

func TestSendSetupModal_ErrorRespond(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		return fmt.Errorf("fail")
	}

	m := &setupManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String(), Type: discordgo.InteractionApplicationCommand, GuildID: "g1"}}
	if err := m.SendSetupModal(context.Background(), i); err == nil {
		t.Fatalf("expected error when InteractionRespond fails")
	}
}

func TestHandleSetupModalSubmit_MissingComponents(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	// Expect an error response (ephemeral message)
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		return nil
	}

	m := &setupManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	// Build modal submit with no components to trigger error path
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      uuid.New().String(),
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "g1",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "guild_setup_modal", Components: []discordgo.MessageComponent{}},
	}}

	if err := m.HandleSetupModalSubmit(context.Background(), i); err != nil {
		t.Fatalf("HandleSetupModalSubmit returned error: %v", err)
	}
}

func TestHandleSetupModalSubmit_GuildFetchError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	// Fail guild lookup, then respond error
	fakeSession.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
		return nil, errors.New("no guild")
	}
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		return nil
	}

	m := &setupManager{
		session:          fakeSession,
		logger:           discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error { return fn(ctx) },
	}

	// Provide 4 components to pass initial validation
	mk := func(id, v string) discordgo.ActionsRow {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{CustomID: id, Value: v}}}
	}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      uuid.New().String(),
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "g1",
		Data:    discordgo.ModalSubmitInteractionData{CustomID: "guild_setup_modal", Components: []discordgo.MessageComponent{mk("channel_prefix", "frolf"), mk("role_names", "P, E, A"), mk("signup_message", "m"), mk("signup_emoji", "🥏")}},
	}}

	if err := m.HandleSetupModalSubmit(context.Background(), i); err != nil {
		t.Fatalf("HandleSetupModalSubmit returned error: %v", err)
	}
}

func TestSendFollowups(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	m := &setupManager{session: fakeSession, logger: discardLogger()}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String()}}

	if err := m.sendFollowupError(i, "oops"); err != nil {
		t.Fatalf("sendFollowupError err: %v", err)
	}
	if err := m.sendFollowupSuccess(i, &SetupResult{}); err != nil {
		t.Fatalf("sendFollowupSuccess err: %v", err)
	}
}

func TestPublishSetupEvent_Success(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	eb := &testutils.FakeEventBus{}

	fakeSession.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
		return &discordgo.Guild{ID: "g1", Name: "Guild"}, nil
	}
	eb.PublishFunc = func(topic string, msgs ...*message.Message) error {
		if topic != guildevents.GuildSetupRequestedV1 {
			return fmt.Errorf("unexpected topic: %s", topic)
		}
		return nil
	}

	m := &setupManager{
		session:   fakeSession,
		publisher: eb,
		logger:    discardLogger(),
		helper:    utils.NewHelper(discardLogger()),
	}

	res := &SetupResult{
		EventChannelID:       "e",
		EventChannelName:     "events",
		LeaderboardChannelID: "l",
		SignupChannelID:      "s",
		RoleMappings:         map[string]string{"Player": "rp", "Editor": "re", "Admin": "ra"},
		UserRoleID:           "rp",
		EditorRoleID:         "re",
		AdminRoleID:          "ra",
		SignupEmoji:          "🥏",
		SignupMessageID:      "mid",
	}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{GuildID: "g1", Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}}}}

	if err := m.publishSetupEvent(i, res); err != nil {
		t.Fatalf("publishSetupEvent error: %v", err)
	}
}

// fakeHelpers lets us inject helper errors
type fakeHelpers struct{}

func (fakeHelpers) CreateResultMessage(_ *message.Message, _ interface{}, _ string) (*message.Message, error) {
	return nil, nil
}

func (fakeHelpers) CreateNewMessage(_ interface{}, _ string) (*message.Message, error) {
	return nil, fmt.Errorf("helper fail")
}
func (fakeHelpers) UnmarshalPayload(_ *message.Message, _ interface{}) error { return nil }

func TestPublishSetupEvent_Errors(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	eb := &testutils.FakeEventBus{}

	// Case 1: missing user role id -> immediate error
	m1 := &setupManager{}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{GuildID: "g1", Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}}}}
	if err := m1.publishSetupEvent(i, &SetupResult{UserRoleID: "", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error for missing user role id")
	}

	// Case 2: helper CreateNewMessage fails
	fakeSession.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
		return &discordgo.Guild{ID: "g1", Name: "Guild"}, nil
	}
	m2 := &setupManager{session: fakeSession, publisher: eb, logger: discardLogger(), helper: fakeHelpers{}}
	if err := m2.publishSetupEvent(i, &SetupResult{UserRoleID: "u", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error when helper fails")
	}

	// Case 3: publish fails
	fakeSession.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
		return &discordgo.Guild{ID: "g1", Name: "Guild"}, nil
	}
	eb.PublishFunc = func(topic string, msgs ...*message.Message) error {
		return fmt.Errorf("pub fail")
	}
	m3 := &setupManager{session: fakeSession, publisher: eb, logger: discardLogger(), helper: utils.NewHelper(discardLogger())}
	if err := m3.publishSetupEvent(i, &SetupResult{UserRoleID: "u", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error when publish fails")
	}
}

func TestCreateOrFindChannel_Existing(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	fakeSession.GuildChannelsFunc = func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
		return []*discordgo.Channel{{ID: "c1", Name: "frolf-events", Type: discordgo.ChannelTypeGuildText}}, nil
	}

	m := &setupManager{session: fakeSession}
	id, err := m.createOrFindChannel("g1", "frolf-events", "📊")
	if err != nil {
		t.Fatalf("createOrFindChannel error: %v", err)
	}
	if id != "c1" {
		t.Fatalf("expected existing channel id c1, got %s", id)
	}
}

func TestCreateOrFindRole_Existing(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	m := &setupManager{session: fakeSession}
	gid := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{{ID: "r1", Name: "Player"}}}

	id, err := m.createOrFindRole(gid, "Player", 0x00ff00)
	if err != nil {
		t.Fatalf("createOrFindRole error: %v", err)
	}
	if id != "r1" {
		t.Fatalf("expected existing role id r1, got %s", id)
	}
}

func TestCreateOrFindRole_EdgeCases(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	m := &setupManager{session: fakeSession}

	// Existing role with empty ID -> error
	gid := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{{ID: "", Name: "Player"}}}
	if _, err := m.createOrFindRole(gid, "Player", 0x00ff00); err == nil {
		t.Fatalf("expected error for empty existing role id")
	}

	// Creation returns empty ID -> error
	gid2 := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{}}
	fakeSession.GuildRoleCreateFunc = func(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
		return &discordgo.Role{Name: "Player", ID: ""}, nil
	}
	if _, err := m.createOrFindRole(gid2, "Player", 0x00ff00); err == nil {
		t.Fatalf("expected error for empty created role id")
	}
}

func TestCreateSignupMessage_DefaultsAndReactionError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	// Send with empty content and emoji to trigger defaults
	fakeSession.ChannelMessageSendFunc = func(cid, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if content == "" {
			t.Fatalf("content should not be empty (default should apply)")
		}
		return &discordgo.Message{ID: "m1"}, nil
	}
	// Reaction add fails, but function should still return message ID
	fakeSession.MessageReactionAddFunc = func(channelID, messageID, emojiID string) error {
		return fmt.Errorf("react fail")
	}

	m := &setupManager{session: fakeSession, logger: discardLogger()}
	mid, err := m.createSignupMessage(context.Background(), "g1", "ch1", "", "")
	if err != nil {
		t.Fatalf("createSignupMessage unexpected error: %v", err)
	}
	if mid != "m1" {
		t.Fatalf("expected message id m1, got %s", mid)
	}
}

func TestCreateSignupMessage_SendError(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return nil, fmt.Errorf("send fail")
	}
	m := &setupManager{session: fakeSession, logger: discardLogger()}
	if _, err := m.createSignupMessage(context.Background(), "g1", "ch1", "m", "🥏"); err == nil {
		t.Fatalf("expected error when ChannelMessageSend fails")
	}
}
