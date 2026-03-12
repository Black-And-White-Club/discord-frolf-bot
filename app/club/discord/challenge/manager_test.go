package challenge

import (
	"context"
	"strings"
	"testing"
	"time"

	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestManagerHandleChallengeCommandOpenPublishesRequest(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var publishedTopic string
	var publishedPayload *clubevents.ChallengeOpenRequestedPayloadV1
	var editedContent string

	fakeHelper.CreateNewMessageFunc = func(payload any, topic string) (*message.Message, error) {
		typed, ok := payload.(*clubevents.ChallengeOpenRequestedPayloadV1)
		if !ok {
			t.Fatalf("unexpected payload type: %T", payload)
		}
		publishedPayload = typed
		return message.NewMessage("msg-1", []byte("{}")), nil
	}
	fakeBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		publishedTopic = topic
		if len(messages) != 1 {
			t.Fatalf("expected one message, got %d", len(messages))
		}
		if got := messages[0].Metadata.Get("guild_id"); got != "guild-1" {
			t.Fatalf("expected guild_id metadata, got %q", got)
		}
		if messages[0].Metadata.Get("correlation_id") == "" {
			t.Fatal("expected correlation_id metadata to be set")
		}
		return nil
	}
	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if edit.Content != nil {
			editedContent = *edit.Content
		}
		return &discordgo.Message{ID: "edited"}, nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "actor-1"}},
			Type:    discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "open",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "user",
								Type:  discordgo.ApplicationCommandOptionUser,
								Value: "target-1",
							},
						},
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if publishedTopic != clubevents.ChallengeOpenRequestedV1 {
		t.Fatalf("expected topic %q, got %q", clubevents.ChallengeOpenRequestedV1, publishedTopic)
	}
	if publishedPayload == nil {
		t.Fatal("expected challenge payload to be captured")
	}
	if publishedPayload.GuildID != "guild-1" || publishedPayload.ActorExternalID != "actor-1" || publishedPayload.TargetExternalID != "target-1" {
		t.Fatalf("unexpected payload: %+v", publishedPayload)
	}
	if !strings.Contains(editedContent, "<@target-1>") {
		t.Fatalf("expected deferred response to mention target, got %q", editedContent)
	}
}

func TestManagerHandleChallengeFactPostsCardAndBindsMessage(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var sentChannelID string
	var bindTopic string
	var bindPayload *clubevents.ChallengeMessageBindRequestedPayloadV1

	fakeResolver.GetGuildConfigFunc = func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
		return &storage.GuildConfig{EventChannelID: "events-1"}, nil
	}
	fakeSession.ChannelMessageSendComplexFunc = func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		sentChannelID = channelID
		if len(data.Embeds) != 1 {
			t.Fatalf("expected one embed, got %d", len(data.Embeds))
		}
		return &discordgo.Message{ID: "message-1", ChannelID: channelID}, nil
	}
	fakeHelper.CreateNewMessageFunc = func(payload any, topic string) (*message.Message, error) {
		typed, ok := payload.(*clubevents.ChallengeMessageBindRequestedPayloadV1)
		if !ok {
			t.Fatalf("unexpected bind payload type: %T", payload)
		}
		bindPayload = typed
		return message.NewMessage("msg-2", []byte("{}")), nil
	}
	fakeBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		bindTopic = topic
		return nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	)

	guildID := "guild-1"
	payload := &clubevents.ChallengeFactPayloadV1{
		Challenge: clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:             "challenge-1",
				DiscordGuildID: &guildID,
				Status:         clubtypes.ChallengeStatusOpen,
				OpenedAt:       time.Now().UTC(),
			},
		},
	}

	if err := manager.HandleChallengeFact(context.Background(), clubevents.ChallengeOpenedV1, payload); err != nil {
		t.Fatalf("HandleChallengeFact returned error: %v", err)
	}

	if sentChannelID != "events-1" {
		t.Fatalf("expected challenge card in events-1, got %q", sentChannelID)
	}
	if bindTopic != clubevents.ChallengeMessageBindRequestedV1 {
		t.Fatalf("expected bind topic %q, got %q", clubevents.ChallengeMessageBindRequestedV1, bindTopic)
	}
	if bindPayload == nil {
		t.Fatal("expected bind payload to be captured")
	}
	if bindPayload.ChallengeID != "challenge-1" || bindPayload.MessageID != "message-1" || bindPayload.ChannelID != "events-1" {
		t.Fatalf("unexpected bind payload: %+v", bindPayload)
	}
}

func TestManagerHandleChallengeFactEditsExistingCardAndAnnouncesLinkedRound(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var editedMessageID string
	var announcement string

	fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		editedMessageID = edit.ID
		return &discordgo.Message{ID: edit.ID, ChannelID: edit.Channel}, nil
	}
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		announcement = content
		return &discordgo.Message{ID: "announce-1", ChannelID: channelID, Content: content}, nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	)

	guildID := "guild-1"
	challengerExternalID := "challenger-1"
	defenderExternalID := "defender-1"
	payload := &clubevents.ChallengeFactPayloadV1{
		Challenge: clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:                   "challenge-2",
				DiscordGuildID:       &guildID,
				Status:               clubtypes.ChallengeStatusAccepted,
				ChallengerUserUUID:   "challenger-uuid",
				DefenderUserUUID:     "defender-uuid",
				ChallengerExternalID: &challengerExternalID,
				DefenderExternalID:   &defenderExternalID,
				OpenedAt:             time.Now().UTC(),
				LinkedRound: &clubtypes.ChallengeRoundLink{
					RoundID:  "round-9",
					LinkedAt: time.Now().UTC(),
					IsActive: true,
				},
			},
			MessageBinding: &clubtypes.ChallengeMessageBinding{
				GuildID:   guildID,
				ChannelID: "events-1",
				MessageID: "message-9",
			},
		},
	}

	if err := manager.HandleChallengeFact(context.Background(), clubevents.ChallengeRoundLinkedV1, payload); err != nil {
		t.Fatalf("HandleChallengeFact returned error: %v", err)
	}

	if editedMessageID != "message-9" {
		t.Fatalf("expected existing card to be edited, got %q", editedMessageID)
	}
	if !strings.Contains(announcement, "Challenge Round Scheduled") || !strings.Contains(announcement, "`round-9`") {
		t.Fatalf("expected round announcement, got %q", announcement)
	}
}

func TestManagerHandleChallengeFactSkipsDuplicateRoundAnnouncement(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var announcementCalls int

	fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{ID: edit.ID, ChannelID: edit.Channel}, nil
	}
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		announcementCalls++
		return &discordgo.Message{ID: "announce-1", ChannelID: channelID, Content: content}, nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	)

	guildID := "guild-1"
	payload := &clubevents.ChallengeFactPayloadV1{
		Challenge: clubtypes.ChallengeDetail{
			ChallengeSummary: clubtypes.ChallengeSummary{
				ID:             "challenge-2",
				DiscordGuildID: &guildID,
				Status:         clubtypes.ChallengeStatusAccepted,
				OpenedAt:       time.Now().UTC(),
				LinkedRound: &clubtypes.ChallengeRoundLink{
					RoundID:  "round-9",
					LinkedAt: time.Now().UTC(),
					IsActive: true,
				},
			},
			MessageBinding: &clubtypes.ChallengeMessageBinding{
				GuildID:   guildID,
				ChannelID: "events-1",
				MessageID: "message-9",
			},
		},
	}

	if err := manager.HandleChallengeFact(context.Background(), clubevents.ChallengeRoundLinkedV1, payload); err != nil {
		t.Fatalf("first HandleChallengeFact returned error: %v", err)
	}
	if err := manager.HandleChallengeFact(context.Background(), clubevents.ChallengeRoundLinkedV1, payload); err != nil {
		t.Fatalf("second HandleChallengeFact returned error: %v", err)
	}

	if announcementCalls != 1 {
		t.Fatalf("expected one challenge round announcement, got %d", announcementCalls)
	}
}

func TestManagerHandleChallengeCommandLinkRejectsInvalidRoundID(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var responseType discordgo.InteractionResponseType
	var responseContent string
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		responseType = response.Type
		if response.Data != nil {
			responseContent = response.Data.Content
		}
		return nil
	}
	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		t.Fatal("did not expect deferred response edit for invalid round ID")
		return nil, nil
	}
	fakeHelper.CreateNewMessageFunc = func(payload any, topic string) (*message.Message, error) {
		t.Fatalf("did not expect publish payload creation for invalid round ID: %T", payload)
		return nil, nil
	}
	fakeBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		t.Fatalf("did not expect publish for invalid round ID")
		return nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	).(*manager)

	manager.getChallengeDetail = func(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error) {
		t.Fatal("did not expect challenge detail lookup for invalid round ID")
		return nil, nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "actor-1"}},
			Type:    discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "link",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "challenge_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "challenge-1",
							},
							{
								Name:  "round_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "not-a-uuid",
							},
						},
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if responseType != discordgo.InteractionResponseChannelMessageWithSource {
		t.Fatalf("expected immediate ephemeral response, got %v", responseType)
	}
	if responseContent != "Provide a valid round ID." {
		t.Fatalf("expected invalid round ID message, got %q", responseContent)
	}
}

func TestManagerHandleChallengeCommandLinkPublishesRequestForValidRoundID(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var publishedTopic string
	var publishedPayload *clubevents.ChallengeRoundLinkRequestedPayloadV1
	var responseType discordgo.InteractionResponseType
	var editedContent string

	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		responseType = response.Type
		return nil
	}
	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if edit.Content != nil {
			editedContent = *edit.Content
		}
		return &discordgo.Message{ID: "edited"}, nil
	}
	fakeHelper.CreateNewMessageFunc = func(payload any, topic string) (*message.Message, error) {
		typed, ok := payload.(*clubevents.ChallengeRoundLinkRequestedPayloadV1)
		if !ok {
			t.Fatalf("unexpected payload type: %T", payload)
		}
		publishedPayload = typed
		return message.NewMessage("msg-1", []byte("{}")), nil
	}
	fakeBus.PublishFunc = func(topic string, messages ...*message.Message) error {
		publishedTopic = topic
		return nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	).(*manager)

	manager.getChallengeDetail = func(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error) {
		challenger := "actor-1"
		return &clubevents.ChallengeDetailResponsePayloadV1{
			Challenge: &clubtypes.ChallengeDetail{
				ChallengeSummary: clubtypes.ChallengeSummary{
					ID:                   challengeID,
					Status:               clubtypes.ChallengeStatusAccepted,
					OpenedAt:             time.Now().UTC(),
					ChallengerExternalID: &challenger,
				},
			},
		}, nil
	}

	validRoundID := uuid.NewString()
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "actor-1"}},
			Type:    discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "link",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "challenge_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "challenge-1",
							},
							{
								Name:  "round_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: validRoundID,
							},
						},
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if responseType != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Fatalf("expected deferred response, got %v", responseType)
	}
	if publishedTopic != clubevents.ChallengeRoundLinkRequestedV1 {
		t.Fatalf("expected topic %q, got %q", clubevents.ChallengeRoundLinkRequestedV1, publishedTopic)
	}
	if publishedPayload == nil {
		t.Fatal("expected link payload to be captured")
	}
	if publishedPayload.RoundID != validRoundID {
		t.Fatalf("expected round ID %q, got %q", validRoundID, publishedPayload.RoundID)
	}
	if editedContent != "Challenge round link requested. The card will update shortly." {
		t.Fatalf("unexpected deferred response content: %q", editedContent)
	}
}

func TestManagerHandleChallengeCommandScheduleOpensCreateRoundModal(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var modalCustomID string
	var modalTitle string
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		if response.Type != discordgo.InteractionResponseModal {
			t.Fatalf("expected modal response, got %v", response.Type)
		}
		modalCustomID = response.Data.CustomID
		modalTitle = response.Data.Title
		return nil
	}

	createRoundManager := createround.NewCreateRoundManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		testutils.NewFakeStorage[any](),
		nil,
		noop.NewTracerProvider().Tracer("test"),
		discordmetrics.NewNoop(),
		fakeResolver,
	)

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		createRoundManager,
	).(*manager)

	manager.getChallengeDetail = func(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error) {
		challenger := "actor-1"
		return &clubevents.ChallengeDetailResponsePayloadV1{
			Challenge: &clubtypes.ChallengeDetail{
				ChallengeSummary: clubtypes.ChallengeSummary{
					ID:                   challengeID,
					Status:               clubtypes.ChallengeStatusAccepted,
					OpenedAt:             time.Now().UTC(),
					ChallengerExternalID: &challenger,
				},
			},
		}, nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "actor-1"}},
			Type:    discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "schedule",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "challenge_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "challenge-1",
							},
						},
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if modalCustomID != createround.ChallengeScheduleModalCustomID("challenge-1") {
		t.Fatalf("expected challenge-aware modal custom ID, got %q", modalCustomID)
	}
	if modalTitle != "Schedule Challenge Round" {
		t.Fatalf("expected schedule modal title, got %q", modalTitle)
	}
}

func TestManagerHandleChallengeCommandScheduleOpensModalWithoutPreloadingChallengeDetail(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var responseType discordgo.InteractionResponseType
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		responseType = response.Type
		return nil
	}

	createRoundManager := createround.NewCreateRoundManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		testutils.NewFakeStorage[any](),
		nil,
		noop.NewTracerProvider().Tracer("test"),
		discordmetrics.NewNoop(),
		fakeResolver,
	)

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{},
		fakeResolver,
		discordmetrics.NewNoop(),
		createRoundManager,
	).(*manager)

	detailLookups := 0
	manager.getChallengeDetail = func(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error) {
		detailLookups++
		return nil, nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member: &discordgo.Member{
				User:  &discordgo.User{ID: "actor-1"},
				Roles: []string{"player"},
			},
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "schedule",
						Type: discordgo.ApplicationCommandOptionSubCommand,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name:  "challenge_id",
								Type:  discordgo.ApplicationCommandOptionString,
								Value: "challenge-1",
							},
						},
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if responseType != discordgo.InteractionResponseModal {
		t.Fatalf("expected modal response, got %v", responseType)
	}
	if detailLookups != 0 {
		t.Fatalf("expected schedule command to avoid preloading challenge detail, got %d lookups", detailLookups)
	}
}

func TestManagerHandleChallengeCommandListRespondsWithLiveSummary(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeBus := &testutils.FakeEventBus{}
	fakeHelper := &testutils.FakeHelpers{}
	fakeResolver := &testutils.FakeGuildConfigResolver{}

	var responseType discordgo.InteractionResponseType
	var editedContent string
	fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, response *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
		responseType = response.Type
		return nil
	}
	fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if edit.Content != nil {
			editedContent = *edit.Content
		}
		return &discordgo.Message{ID: "edited"}, nil
	}

	manager := NewManager(
		fakeSession,
		fakeBus,
		testutils.NoOpLogger(),
		fakeHelper,
		&config.Config{PWA: config.PWAConfig{BaseURL: "https://app.example"}},
		fakeResolver,
		discordmetrics.NewNoop(),
		nil,
	).(*manager)

	manager.listChallenges = func(ctx context.Context, guildID string, statuses []clubtypes.ChallengeStatus) (*clubevents.ChallengeListResponsePayloadV1, error) {
		challenger := "challenger-1"
		defender := "defender-1"
		return &clubevents.ChallengeListResponsePayloadV1{
			Challenges: []clubtypes.ChallengeSummary{
				{
					ID:                   "challenge-1",
					Status:               clubtypes.ChallengeStatusOpen,
					ChallengerUserUUID:   "challenger-uuid",
					DefenderUserUUID:     "defender-uuid",
					ChallengerExternalID: &challenger,
					DefenderExternalID:   &defender,
					OpenedAt:             time.Now().UTC(),
				},
			},
		}, nil
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			GuildID: "guild-1",
			Member:  &discordgo.Member{User: &discordgo.User{ID: "actor-1"}},
			Type:    discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "challenge",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "list",
						Type: discordgo.ApplicationCommandOptionSubCommand,
					},
				},
			},
		},
	}

	if err := manager.HandleChallengeCommand(context.Background(), interaction); err != nil {
		t.Fatalf("HandleChallengeCommand returned error: %v", err)
	}

	if responseType != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Fatalf("expected deferred ephemeral response, got %v", responseType)
	}
	if !strings.Contains(editedContent, "Active challenges: 1") {
		t.Fatalf("expected summary count, got %q", editedContent)
	}
	if !strings.Contains(editedContent, "`challeng` open: <@challenger-1> vs <@defender-1>") {
		t.Fatalf("expected compact challenge summary, got %q", editedContent)
	}
	if !strings.Contains(editedContent, "Open in App: https://app.example/challenges") {
		t.Fatalf("expected app link in summary, got %q", editedContent)
	}
}

func TestFormatChallengeListSummary_SortsOpenFirstAndLimitsResults(t *testing.T) {
	now := time.Now().UTC()
	challenges := make([]clubtypes.ChallengeSummary, 0, 12)

	for idx := 0; idx < 6; idx++ {
		challenger := "open-challenger"
		defender := "open-defender"
		challenges = append(challenges, clubtypes.ChallengeSummary{
			ID:                   "open-" + string(rune('a'+idx)),
			Status:               clubtypes.ChallengeStatusOpen,
			OpenedAt:             now.Add(-time.Duration(idx) * time.Minute),
			ChallengerExternalID: &challenger,
			DefenderExternalID:   &defender,
		})
	}

	for idx := 0; idx < 6; idx++ {
		challenger := "accepted-challenger"
		defender := "accepted-defender"
		challenges = append(challenges, clubtypes.ChallengeSummary{
			ID:                   "accepted-" + string(rune('a'+idx)),
			Status:               clubtypes.ChallengeStatusAccepted,
			OpenedAt:             now.Add(-time.Duration(idx) * time.Hour),
			ChallengerExternalID: &challenger,
			DefenderExternalID:   &defender,
		})
	}

	content := formatChallengeListSummary(&config.Config{PWA: config.PWAConfig{BaseURL: "https://app.example"}}, &clubevents.ChallengeListResponsePayloadV1{
		Challenges: challenges,
	})

	lines := strings.Split(content, "\n")
	if len(lines) != 13 {
		t.Fatalf("expected 13 lines (count + 10 entries + overflow + app link), got %d: %q", len(lines), content)
	}
	if !strings.Contains(lines[1], "open") {
		t.Fatalf("expected first listed challenge to be open, got %q", lines[1])
	}
	if lines[11] != "+2 more in app" {
		t.Fatalf("expected overflow line, got %q", lines[11])
	}
	if lines[12] != "Open in App: https://app.example/challenges" {
		t.Fatalf("expected app link line, got %q", lines[12])
	}
}
