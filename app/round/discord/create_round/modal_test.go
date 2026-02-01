package createround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
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
				interactionStore: &testutils.FakeStorage[any]{},
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
			fakeStorage := &testutils.FakeStorage[any]{}
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
			fakeStorage := &testutils.FakeStorage[any]{}
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

func createTestInteraction(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
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
				CustomID: "create_round_modal",
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
