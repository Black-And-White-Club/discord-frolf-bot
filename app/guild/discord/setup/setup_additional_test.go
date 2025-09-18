package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	sharedmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestSendSetupModal_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	// Expect a modal response
	ms.EXPECT().InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(inter *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
			if resp == nil || resp.Type != discordgo.InteractionResponseModal {
				t.Fatalf("expected modal response, got: %#v", resp)
			}
			return nil
		},
	)

	m := &setupManager{
		session: ms,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	ms.EXPECT().InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("fail"))

	m := &setupManager{
		session: ms,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	// Expect an error response (ephemeral message)
	ms.EXPECT().InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	m := &setupManager{
		session: ms,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	// Fail guild lookup, then respond error
	ms.EXPECT().Guild("g1", gomock.Any()).Return(nil, errors.New("no guild"))
	ms.EXPECT().InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	m := &setupManager{
		session:          ms,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	// Error followup
	ms.EXPECT().FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).Return(&discordgo.Message{ID: "e1"}, nil)
	// Success followup
	ms.EXPECT().FollowupMessageCreate(gomock.Any(), true, gomock.Any(), gomock.Any()).Return(&discordgo.Message{ID: "s1"}, nil)

	m := &setupManager{session: ms, logger: discardLogger()}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{ID: uuid.New().String()}}

	if err := m.sendFollowupError(i, "oops"); err != nil {
		t.Fatalf("sendFollowupError err: %v", err)
	}
	if err := m.sendFollowupSuccess(i, &SetupResult{}); err != nil {
		t.Fatalf("sendFollowupSuccess err: %v", err)
	}
}

func TestPublishSetupEvent_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)
	eb := sharedmocks.NewMockEventBus(ctrl)

	ms.EXPECT().Guild("g1", gomock.Any()).Return(&discordgo.Guild{ID: "g1", Name: "Guild"}, nil)
	eb.EXPECT().Publish(guildevents.GuildSetupEventTopic, gomock.Any()).Return(nil)

	m := &setupManager{
		session:   ms,
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)
	eb := sharedmocks.NewMockEventBus(ctrl)

	// Case 1: missing user role id -> immediate error
	m1 := &setupManager{}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{GuildID: "g1", Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}}}}
	if err := m1.publishSetupEvent(i, &SetupResult{UserRoleID: "", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error for missing user role id")
	}

	// Case 2: helper CreateNewMessage fails
	ms.EXPECT().Guild("g1", gomock.Any()).Return(&discordgo.Guild{ID: "g1", Name: "Guild"}, nil)
	m2 := &setupManager{session: ms, publisher: eb, logger: discardLogger(), helper: fakeHelpers{}}
	if err := m2.publishSetupEvent(i, &SetupResult{UserRoleID: "u", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error when helper fails")
	}

	// Case 3: publish fails
	ms.EXPECT().Guild("g1", gomock.Any()).Return(&discordgo.Guild{ID: "g1", Name: "Guild"}, nil)
	eb.EXPECT().Publish(guildevents.GuildSetupEventTopic, gomock.Any()).Return(fmt.Errorf("pub fail"))
	m3 := &setupManager{session: ms, publisher: eb, logger: discardLogger(), helper: utils.NewHelper(discardLogger())}
	if err := m3.publishSetupEvent(i, &SetupResult{UserRoleID: "u", EditorRoleID: "e", AdminRoleID: "a"}); err == nil {
		t.Fatalf("expected error when publish fails")
	}
}

func TestCreateOrFindChannel_Existing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	ms.EXPECT().GuildChannels("g1", gomock.Any()).Return([]*discordgo.Channel{{ID: "c1", Name: "frolf-events", Type: discordgo.ChannelTypeGuildText}}, nil)

	m := &setupManager{session: ms}
	id, err := m.createOrFindChannel("g1", "frolf-events", "📊")
	if err != nil {
		t.Fatalf("createOrFindChannel error: %v", err)
	}
	if id != "c1" {
		t.Fatalf("expected existing channel id c1, got %s", id)
	}
}

func TestCreateOrFindRole_Existing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	m := &setupManager{session: ms}
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)
	m := &setupManager{session: ms}

	// Existing role with empty ID -> error
	gid := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{{ID: "", Name: "Player"}}}
	if _, err := m.createOrFindRole(gid, "Player", 0x00ff00); err == nil {
		t.Fatalf("expected error for empty existing role id")
	}

	// Creation returns empty ID -> error
	gid2 := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{}}
	ms.EXPECT().GuildRoleCreate("g1", gomock.Any(), gomock.Any()).Return(&discordgo.Role{Name: "Player", ID: ""}, nil)
	if _, err := m.createOrFindRole(gid2, "Player", 0x00ff00); err == nil {
		t.Fatalf("expected error for empty created role id")
	}
}

func TestCreateSignupMessage_DefaultsAndReactionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)

	// Send with empty content and emoji to trigger defaults
	ms.EXPECT().ChannelMessageSend("ch1", gomock.Any(), gomock.Any()).DoAndReturn(
		func(cid, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if content == "" {
				t.Fatalf("content should not be empty (default should apply)")
			}
			return &discordgo.Message{ID: "m1"}, nil
		},
	)
	// Reaction add fails, but function should still return message ID
	ms.EXPECT().MessageReactionAdd("ch1", "m1", gomock.Any()).Return(fmt.Errorf("react fail")).AnyTimes()

	m := &setupManager{session: ms, logger: discardLogger()}
	mid, err := m.createSignupMessage("g1", "ch1", "", "")
	if err != nil {
		t.Fatalf("createSignupMessage unexpected error: %v", err)
	}
	if mid != "m1" {
		t.Fatalf("expected message id m1, got %s", mid)
	}
}

func TestCreateSignupMessage_SendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ms := discordmocks.NewMockSession(ctrl)
	ms.EXPECT().ChannelMessageSend("ch1", gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("send fail"))
	m := &setupManager{session: ms, logger: discardLogger()}
	if _, err := m.createSignupMessage("g1", "ch1", "m", "🥏"); err == nil {
		t.Fatalf("expected error when ChannelMessageSend fails")
	}
}
