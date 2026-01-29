package createround

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}

func Test_createRoundManager_HandleCreateRoundCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupSession    func(f *discord.FakeSession)
		expectSuccess   string
		expectErrSubstr string
	}{
		{
			name: "modal sent successfully",
			setupSession: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			expectSuccess: "modal sent",
		},
		{
			name: "modal returns operation error",
			setupSession: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("modal failed")
				}
			},
			expectErrSubstr: "modal failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}

			crm := &createRoundManager{
				session: fakeSession,
				logger:  slog.Default(),
				operationWrapper: func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return operationFunc(ctx)
				},
			}

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			}

			result, _ := crm.HandleCreateRoundCommand(context.Background(), interaction)

			if tt.expectErrSubstr != "" {
				if result.Error == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectErrSubstr)
				} else if !strings.Contains(result.Error.Error(), tt.expectErrSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectErrSubstr, result.Error.Error())
				}
			} else {
				if result.Error != nil {
					t.Errorf("unexpected error: %v", result.Error)
				}
				if result.Success != tt.expectSuccess {
					t.Errorf("unexpected success value: got %v, want %q", result.Success, tt.expectSuccess)
				}
			}
		})
	}
}

func Test_createRoundManager_HandleRetryCreateRound(t *testing.T) {
	tests := []struct {
		name                        string
		setupSession                func(f *discord.FakeSession)
		expectedErrMsg              string
		expectedSuccess             string
		expectInteractionEditCalled bool
	}{
		{
			name: "retry modal sent successfully",
			setupSession: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			expectedSuccess:             "retry modal sent",
			expectInteractionEditCalled: false,
		},
		{
			name: "SendCreateRoundModal returns error",
			setupSession: func(f *discord.FakeSession) {
				f.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("modal failed")
				}
			},
			expectedErrMsg:              "modal failed",
			expectInteractionEditCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			var editCalled bool
			fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
				editCalled = true
				return &discordgo.Message{}, nil
			}

			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}

			crm := &createRoundManager{
				session: fakeSession,
				logger:  slog.Default(),
				operationWrapper: func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return operationFunc(ctx)
				},
			}

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			}

			result, err := crm.HandleRetryCreateRound(context.Background(), interaction)

			if tt.expectedErrMsg != "" {
				if err != nil {
					if !strings.Contains(err.Error(), tt.expectedErrMsg) {
						t.Errorf("expected error to contain %q, got %q", tt.expectedErrMsg, err.Error())
					}
				} else if result.Error == nil {
					t.Errorf("expected result.Error containing %q, got nil", tt.expectedErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.expectedErrMsg) {
					t.Errorf("expected result.Error to contain %q, got %q", tt.expectedErrMsg, result.Error.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if result.Error != nil {
					t.Errorf("expected no result.Error, got %v", result.Error)
				}
				if result.Success != tt.expectedSuccess {
					t.Errorf("expected success: %q, got: %v", tt.expectedSuccess, result.Success)
				}
			}

			if tt.expectInteractionEditCalled != editCalled {
				t.Errorf("InteractionResponseEdit call mismatch: got %v, want %v", editCalled, tt.expectInteractionEditCalled)
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponse(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	logger := slog.Default()
	fakeHelper := &testutils.FakeHelpers{}
	mockConfig := &config.Config{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &testutils.FakeDiscordMetrics{}

	rm := &createRoundManager{
		session:          fakeSession,
		publisher:        nil,
		logger:           logger,
		helper:           fakeHelper,
		config:           mockConfig,
		interactionStore: fakeInteractionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: testOperationWrapper,
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context)
		args  struct {
			ctx           context.Context
			correlationID string
			message       string
			edit          []*discordgo.WebhookEdit
		}
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful update without existing edit",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "int-id"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
				fakeInteractionStore.DeleteFunc = func(ctx context.Context, key string) {
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "corr-123",
				message:       "Updated message",
				edit:          nil,
			},
			wantSuccess: "interaction response updated",
		},
		{
			name: "successful update with existing edit",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "int-id-2"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
				fakeInteractionStore.DeleteFunc = func(ctx context.Context, key string) {
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "corr-456",
				message:       "New message",
				edit: []*discordgo.WebhookEdit{
					{
						Embeds: &[]*discordgo.MessageEmbed{{Title: "Old Embed"}},
					},
				},
			},
			wantSuccess: "interaction response updated",
		},
		{
			name: "interaction not found",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("not found")
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "corr-not-found",
				message:       "This won't be sent",
				edit:          nil,
			},
			wantErrMsg: "no interaction found for correlation ID: corr-not-found",
		},
		{
			name: "interaction response edit fails",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "int-fail"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("discord API error")
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "corr-fail",
				message:       "Failed update",
				edit:          nil,
			},
			wantErrMsg: "discord API error",
		},
		{
			name: "context cancelled",
			setup: func(ctx context.Context) {
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				correlationID: "corr-cancelled",
				message:       "Should not be processed",
				edit:          nil,
			},
			wantErrMsg: context.Canceled.Error(),
			wantErrIs:  context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.args.ctx)

			result, _ := rm.UpdateInteractionResponse(tt.args.ctx, tt.args.correlationID, tt.args.message, tt.args.edit...)

			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("UpdateInteractionResponse() Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
					t.Errorf("Error type mismatch: got %T, want %T", result.Error, tt.wantErrIs)
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponseWithRetryButton(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeInteractionStore := &testutils.FakeStorage[any]{}
	logger := slog.Default()
	fakeHelper := &testutils.FakeHelpers{}
	mockConfig := &config.Config{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &testutils.FakeDiscordMetrics{}

	rm := &createRoundManager{
		session:          fakeSession,
		publisher:        nil,
		logger:           logger,
		helper:           fakeHelper,
		config:           mockConfig,
		interactionStore: fakeInteractionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: testOperationWrapper,
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context)
		args  struct {
			ctx           context.Context
			correlationID string
			message       string
		}
		wantSuccess string
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "successful update with retry button",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "int-id"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "corr-123",
				message:       "Something went wrong",
			},
			wantSuccess: "response updated with retry",
		},
		{
			name: "interaction not found",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("not found")
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "corr-not-found",
				message:       "This won't be sent",
			},
			wantErrMsg: "no interaction found for correlation ID: corr-not-found",
		},
		{
			name: "interaction response edit fails",
			setup: func(ctx context.Context) {
				fakeInteractionStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return &discordgo.Interaction{ID: "int-fail"}, nil
				}
				fakeSession.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("discord API error")
				}
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "corr-fail",
				message:       "Failed update",
			},
			wantErrMsg: "discord API error",
		},
		{
			name: "context cancelled",
			setup: func(ctx context.Context) {
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				correlationID: "corr-cancelled",
				message:       "Should not be processed",
			},
			wantErrMsg: context.Canceled.Error(),
			wantErrIs:  context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.args.ctx)

			result, _ := rm.UpdateInteractionResponseWithRetryButton(tt.args.ctx, tt.args.correlationID, tt.args.message)

			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("UpdateInteractionResponseWithRetryButton() Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
					t.Errorf("Error type mismatch: got %T, want %T", result.Error, tt.wantErrIs)
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
			}
		})
	}
}
