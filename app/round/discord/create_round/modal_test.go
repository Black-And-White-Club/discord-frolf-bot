package createround

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test_createRoundManager_SendCreateRoundModal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(f *discord.FakeSession)
		ctx         context.Context
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful send",
			setup: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					if r.Type != discordgo.InteractionResponseModal {
						t.Errorf("Expected InteractionResponseModal, got %v", r.Type)
					}
					if r.Data.Title != "Create Round" {
						t.Errorf("Expected title 'Create Round', got %v", r.Data.Title)
					}
					return nil
				}
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantSuccess: "modal sent",
		},
		{
			name: "failed to send modal",
			setup: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to send modal")
				}
			},
			ctx: context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantErrMsg: "failed to send create round modal: failed to send modal",
		},
		{
			name:       "nil interaction",
			ctx:        context.Background(),
			args:       nil,
			wantErrMsg: "interaction is nil or incomplete",
		},
		{
			name: "context cancelled before operation",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "user-123"},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantErrMsg: context.Canceled.Error(),
			wantErrIs:  context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			if tt.setup != nil {
				tt.setup(fakeSession)
			}

			crm := &createRoundManager{
				session:          fakeSession,
				logger:           testutils.NoOpLogger(),
				interactionStore: testutils.NewFakeStorage[any](),
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &testutils.FakeDiscordMetrics{},
				operationWrapper: testOperationWrapper,
			}

			result, err := crm.SendCreateRoundModal(tt.ctx, tt.args)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

func Test_createRoundManager_SendCreateRoundModal_UsesContextModalConfig(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		if r.Type != discordgo.InteractionResponseModal {
			t.Fatalf("expected modal response, got %v", r.Type)
		}
		if r.Data.Title != "Schedule Challenge Round" {
			t.Fatalf("expected custom title, got %q", r.Data.Title)
		}
		if r.Data.CustomID != ChallengeScheduleModalCustomID("challenge-1") {
			t.Fatalf("expected challenge custom ID, got %q", r.Data.CustomID)
		}
		return nil
	}

	crm := &createRoundManager{
		session:          fakeSession,
		logger:           testutils.NoOpLogger(),
		interactionStore: testutils.NewFakeStorage[any](),
		tracer:           noop.NewTracerProvider().Tracer("test"),
		metrics:          &testutils.FakeDiscordMetrics{},
		operationWrapper: testOperationWrapper,
	}

	_, err := crm.SendCreateRoundModal(
		WithModalConfig(context.Background(), ModalConfig{
			CustomID: ChallengeScheduleModalCustomID("challenge-1"),
			Title:    "Schedule Challenge Round",
		}),
		&discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID:   "interaction-id",
				Type: discordgo.InteractionApplicationCommand,
				User: &discordgo.User{ID: "user-123"},
				Member: &discordgo.Member{
					User: &discordgo.User{ID: "user-123"},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("SendCreateRoundModal() error = %v", err)
	}
}

func Test_createRoundManager_HandleCreateRoundModalSubmit(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.InteractionCreate
		ctx         context.Context
		setup       func(f *discord.FakeSession, p *testutils.FakeEventBus, s *testutils.FakeStorage[any])
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name:        "successful submission",
			interaction: createTestInteraction("Test Round", "Fun round description", "2025-04-01 14:00", "America/Chicago", "Disc Golf Park"),
			setup: func(f *discord.FakeSession, p *testutils.FakeEventBus, s *testutils.FakeStorage[any]) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
				s.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				p.PublishFunc = func(topic string, messages ...*message.Message) error {
					return nil
				}
			},
			wantSuccess: "round creation request published",
		},
		{
			name:        "missing required fields",
			interaction: createTestInteraction("", "", "", "", ""),
			setup: func(f *discord.FakeSession, p *testutils.FakeEventBus, s *testutils.FakeStorage[any]) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			wantErrMsg: "validation failed: Title is required. Start Time is required.",
		},
		{
			name:        "failed to publish event",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func(f *discord.FakeSession, p *testutils.FakeEventBus, s *testutils.FakeStorage[any]) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
				s.SetFunc = func(ctx context.Context, key string, value any) error {
					return nil
				}
				p.PublishFunc = func(topic string, messages ...*message.Message) error {
					return errors.New("failed to publish")
				}
			},
			wantErrMsg: "failed to publish event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			fakePublisher := &testutils.FakeEventBus{}
			fakeStorage := testutils.NewFakeStorage[any]()
			if tt.setup != nil {
				tt.setup(fakeSession, fakePublisher, fakeStorage)
			}

			crm := &createRoundManager{
				session:          fakeSession,
				publisher:        fakePublisher,
				logger:           testutils.NoOpLogger(),
				helper:           &testutils.FakeHelpers{},
				config:           &config.Config{Discord: config.DiscordConfig{GuildID: "test-guild"}},
				interactionStore: fakeStorage,
				operationWrapper: testOperationWrapper,
			}

			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx
			}

			result, err := crm.HandleCreateRoundModalSubmit(ctx, tt.interaction)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

func Test_createRoundManager_HandleCreateRoundModalCancel(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(f *discord.FakeSession, s *testutils.FakeStorage[any])
		args        *discordgo.InteractionCreate
		wantSuccess string
		wantErrMsg  string
		ctx         context.Context
	}{
		{
			name: "successful_cancel",
			setup: func(f *discord.FakeSession, s *testutils.FakeStorage[any]) {
				s.DeleteFunc = func(ctx context.Context, key string) {
				}
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
				},
			},
			wantSuccess: "round creation cancelled",
			ctx:         context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			fakeStorage := testutils.NewFakeStorage[any]()
			if tt.setup != nil {
				tt.setup(fakeSession, fakeStorage)
			}

			crm := &createRoundManager{
				session:          fakeSession,
				interactionStore: fakeStorage,
				logger:           testutils.NoOpLogger(),
				operationWrapper: testOperationWrapper,
			}

			result, err := crm.HandleCreateRoundModalCancel(tt.ctx, tt.args)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

func Test_createRoundManager_HandleCreateRoundModalSubmit_PublishesChallengeIDInPayload(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}
	fakeStorage := testutils.NewFakeStorage[any]()

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		return nil
	}
	fakeStorage.SetFunc = func(ctx context.Context, key string, value any) error {
		return nil
	}

	var publishedPayload discordroundevents.CreateRoundModalPayloadV1
	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		if topic != discordroundevents.RoundCreateModalSubmittedV1 {
			t.Fatalf("unexpected topic %q", topic)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		if err := json.Unmarshal(messages[0].Payload, &publishedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		return nil
	}

	crm := &createRoundManager{
		session:          fakeSession,
		publisher:        fakePublisher,
		logger:           testutils.NoOpLogger(),
		helper:           &testutils.FakeHelpers{},
		config:           &config.Config{},
		interactionStore: fakeStorage,
		operationWrapper: testOperationWrapper,
		challengeValidator: func(ctx context.Context, i *discordgo.InteractionCreate, challengeID string) error {
			return nil
		},
	}

	interaction := createTestInteractionWithCustomID(
		ChallengeScheduleModalCustomID("challenge-22"),
		"Test Round",
		"Fun round description",
		"2025-04-01 14:00",
		"America/Chicago",
		"Disc Golf Park",
	)
	if got := interaction.ModalSubmitData().CustomID; got != ChallengeScheduleModalCustomID("challenge-22") {
		t.Fatalf("expected modal custom ID %q, got %q", ChallengeScheduleModalCustomID("challenge-22"), got)
	}
	if got := challengeScheduleIDFromCustomID(interaction.ModalSubmitData().CustomID); got != "challenge-22" {
		t.Fatalf("expected parsed challenge ID challenge-22, got %q", got)
	}

	if _, err := crm.HandleCreateRoundModalSubmit(context.Background(), interaction); err != nil {
		t.Fatalf("HandleCreateRoundModalSubmit() error = %v", err)
	}

	if publishedPayload.ChallengeID == nil || *publishedPayload.ChallengeID != "challenge-22" {
		t.Fatalf("expected challenge_id challenge-22, got %+v", publishedPayload.ChallengeID)
	}
}

func Test_createRoundManager_HandleCreateRoundModalSubmit_RejectsInvalidChallengeScheduleAfterAck(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}
	fakeStorage := testutils.NewFakeStorage[any]()

	var ackContent string
	var editedContent string
	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		if r.Data != nil {
			ackContent = r.Data.Content
		}
		return nil
	}
	fakeSession.InteractionResponseEditFunc = func(i *discordgo.Interaction, edit *discordgo.WebhookEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
		if edit.Content != nil {
			editedContent = *edit.Content
		}
		return &discordgo.Message{ID: "edited"}, nil
	}
	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		t.Fatal("expected modal submit to stop before publishing round request")
		return nil
	}

	crm := &createRoundManager{
		session:          fakeSession,
		publisher:        fakePublisher,
		logger:           testutils.NoOpLogger(),
		helper:           &testutils.FakeHelpers{},
		config:           &config.Config{},
		interactionStore: fakeStorage,
		operationWrapper: testOperationWrapper,
		challengeValidator: func(ctx context.Context, i *discordgo.InteractionCreate, challengeID string) error {
			if challengeID != "challenge-44" {
				t.Fatalf("expected challenge-44, got %q", challengeID)
			}
			return errors.New("Only accepted challenges can schedule a round.")
		},
	}

	if _, err := crm.HandleCreateRoundModalSubmit(context.Background(), createTestInteractionWithCustomID(
		ChallengeScheduleModalCustomID("challenge-44"),
		"Test Round",
		"Fun round description",
		"2025-04-01 14:00",
		"America/Chicago",
		"Disc Golf Park",
	)); err != nil {
		t.Fatalf("HandleCreateRoundModalSubmit() error = %v", err)
	}

	if ackContent != "Round creation request received" {
		t.Fatalf("expected immediate ack, got %q", ackContent)
	}
	if editedContent != "❌ Round creation failed: Only accepted challenges can schedule a round." {
		t.Fatalf("unexpected edited content: %q", editedContent)
	}
}

func createTestInteraction(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
	return createTestInteractionWithCustomID(defaultCreateRoundModalID, title, description, startTime, timezone, location)
}

func createTestInteractionWithCustomID(customID, title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:      "interaction-id",
			Token:   "interaction-token",
			GuildID: "test-guild",
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user-123"},
			},
			Type: discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: customID,
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: "title", Value: title}}},
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: "description", Value: description}}},
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: "start_time", Value: startTime}}},
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: "timezone", Value: timezone}}},
					&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.TextInput{CustomID: "location", Value: location}}},
				},
			},
		},
	}
}
