package updateround

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
)

func Test_updateRoundManager_SendupdateRoundModal(t *testing.T) {
	tests := []struct {
		name         string
		setupSession func(fs *discord.FakeSession)
		ctx          context.Context
		args         *discordgo.InteractionCreate
		wantSuccess  string
		wantErrMsg   string
		wantErrIs    error
	}{
		{
			name: "successful send",
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					if r.Type != discordgo.InteractionResponseModal {
						t.Errorf("Expected InteractionResponseModal, got %v", r.Type)
					}
					if r.Data.Title != "Update Round" {
						t.Errorf("Expected title 'Update Round', got %v", r.Data.Title)
					}
					expectedCustomIDPrefix := "update_round_modal|"
					if !strings.HasPrefix(r.Data.CustomID, expectedCustomIDPrefix) {
						t.Errorf("Expected CustomID to start with '%s', got %v", expectedCustomIDPrefix, r.Data.CustomID)
					}
					if len(r.Data.Components) != 5 {
						t.Errorf("Expected 5 components, got %d", len(r.Data.Components))
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
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "failed to send modal",
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
			wantSuccess: "",
			wantErrMsg:  "failed to send modal",
			wantErrIs:   nil,
		},
		{
			name:         "nil interaction",
			setupSession: func(fs *discord.FakeSession) {},
			ctx:          context.Background(),
			args:         nil,
			wantSuccess:  "",
			wantErrMsg:   "interaction is nil or incomplete",
			wantErrIs:    nil,
		},
		{
			name:         "nil user in interaction",
			setupSession: func(fs *discord.FakeSession) {},
			ctx:          context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionApplicationCommand,
					User: nil,
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name:         "nil member and user in interaction",
			setupSession: func(fs *discord.FakeSession) {},
			ctx:          context.Background(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:     "interaction-id",
					Type:   discordgo.InteractionApplicationCommand,
					User:   nil,
					Member: nil,
				},
			},
			wantSuccess: "",
			wantErrMsg:  "user ID is missing",
			wantErrIs:   nil,
		},
		{
			name:         "context cancelled before operation",
			setupSession: func(fs *discord.FakeSession) {},
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
			wantSuccess: "",
			wantErrMsg:  context.Canceled.Error(),
			wantErrIs:   context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			mockLogger := loggerfrolfbot.NoOpLogger
			fakeInteractionStore := testutils.NewFakeStorage[any]()
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}

			urm := &updateRoundManager{
				session:          fakeSession,
				logger:           mockLogger,
				interactionStore: fakeInteractionStore,
				tracer:           tracer,
				metrics:          metrics,
				operationWrapper: testOperationWrapper,
			}

			testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
			testRoundID := sharedtypes.RoundID(testUUID)
			result, err := urm.SendUpdateRoundModal(tt.ctx, tt.args, testRoundID)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("SendupdateRoundModal() expected error containing %q, but got nil", tt.wantErrMsg)
					t.FailNow()
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SendupdateRoundModal() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("SendupdateRoundModal() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("SendupdateRoundModal() unexpected error: %v", err)
					t.FailNow()
				}
			}

			if tt.wantErrMsg == "" {
				gotSuccess, _ := result.Success.(string)
				if gotSuccess != tt.wantSuccess {
					t.Errorf("SendupdateRoundModal() UpdateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			}
		})
	}
}

// Helper function to create a test interaction with modal submit data for update round
func createTestUpdateInteraction(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:      "interaction-id",
			Token:   "interaction-token",
			GuildID: "test-guild",
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
			Type: discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: "update_round_modal|550e8400-e29b-41d4-a716-446655440000|message-123",
				Components: []discordgo.MessageComponent{
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "title",
								Value:    title,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "description",
								Value:    description,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "start_time",
								Value:    startTime,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "timezone",
								Value:    timezone,
							},
						},
					},
					&discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							&discordgo.TextInput{
								CustomID: "location",
								Value:    location,
							},
						},
					},
				},
			},
		},
	}
}

func Test_updateRoundManager_HandleUpdateRoundModalSubmit(t *testing.T) {
	tests := []struct {
		name           string
		interaction    *discordgo.InteractionCreate
		ctx            context.Context
		setupSession   func(fs *discord.FakeSession)
		setupPublisher func(fp *testutils.FakeEventBus)
		wantSuccess    string
		wantErrMsg     string
		wantErrIs      error
	}{
		{
			name:        "successful submission",
			interaction: createTestUpdateInteraction("Updated Round", "Updated description", "2025-05-01 15:00", "America/Chicago", "Updated Location"),
			ctx:         context.Background(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {
				fp.PublishFunc = func(topic string, msgs ...*message.Message) error {
					if topic != discordroundevents.RoundUpdateModalSubmittedV1 {
						t.Errorf("expected topic %s, got %s", discordroundevents.RoundUpdateModalSubmittedV1, topic)
					}
					return nil
				}
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "missing required fields",
			interaction: createTestUpdateInteraction("", "", "", "", ""),
			ctx:         context.Background(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					if !strings.Contains(r.Data.Content, "Please fill at least one field") {
						t.Errorf("Expected error message to contain 'Please fill at least one field', got %v", r.Data.Content)
					}
					return nil
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {},
			wantSuccess:    "",
			wantErrMsg:     "no fields provided",
			wantErrIs:      nil,
		},
		{
			name:        "field too long",
			interaction: createTestUpdateInteraction(strings.Repeat("A", 101), "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {
				fp.PublishFunc = func(topic string, msgs ...*message.Message) error {
					return nil
				}
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "failed to acknowledge submission",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to acknowledge")
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {
				fp.PublishFunc = func(topic string, msgs ...*message.Message) error {
					return nil
				}
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:        "failed to publish event",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx:         context.Background(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {
				fp.PublishFunc = func(topic string, msgs ...*message.Message) error {
					return errors.New("failed to publish")
				}
			},
			wantSuccess: "",
			wantErrMsg:  "failed to publish",
			wantErrIs:   nil,
		},
		{
			name:           "nil interaction",
			interaction:    nil,
			ctx:            context.Background(),
			setupSession:   func(fs *discord.FakeSession) {},
			setupPublisher: func(fp *testutils.FakeEventBus) {},
			wantSuccess:    "",
			wantErrMsg:     "interaction is nil or incomplete",
			wantErrIs:      nil,
		},
		{
			name: "missing user ID",
			interaction: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					Type: discordgo.InteractionModalSubmit,
					Data: discordgo.ModalSubmitInteractionData{},
				},
			},
			ctx:            context.Background(),
			setupSession:   func(fs *discord.FakeSession) {},
			setupPublisher: func(fp *testutils.FakeEventBus) {},
			wantSuccess:    "",
			wantErrMsg:     "user ID is missing",
			wantErrIs:      nil,
		},
		{
			name:        "context cancelled",
			interaction: createTestUpdateInteraction("Updated Round", "Description", "2025-05-01 15:00", "UTC", "Location"),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return nil
				}
			},
			setupPublisher: func(fp *testutils.FakeEventBus) {
				fp.PublishFunc = func(topic string, msgs ...*message.Message) error {
					return nil
				}
			},
			wantSuccess: "round update request published",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			fakePublisher := &testutils.FakeEventBus{}
			fakeInteractionStore := testutils.NewFakeStorage[any]()
			mockLogger := loggerfrolfbot.NoOpLogger
			fakeHelper := &testutils.FakeHelpers{}

			tt.setupSession(fakeSession)
			tt.setupPublisher(fakePublisher)

			urm := &updateRoundManager{
				session:          fakeSession,
				publisher:        fakePublisher,
				logger:           mockLogger,
				helper:           fakeHelper,
				config:           &config.Config{},
				interactionStore: fakeInteractionStore,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
				operationWrapper: testOperationWrapper,
			}

			result, err := urm.HandleUpdateRoundModalSubmit(tt.ctx, tt.interaction)

			gotSuccess, ok := result.Success.(string)
			if ok {
				if gotSuccess != tt.wantSuccess {
					t.Errorf("HandleUpdateRoundModalSubmit() UpdateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
				}
			} else if tt.wantSuccess != "" {
				t.Errorf("HandleUpdateRoundModalSubmit() UpdateRoundOperationResult.Success was not a string: got %T, want string", result.Success)
			}

			if tt.wantErrMsg != "" {
				actualErr := err
				if actualErr == nil && result.Error != nil {
					actualErr = result.Error
				}
				if actualErr == nil {
					t.Errorf("HandleUpdateRoundModalSubmit() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(actualErr.Error(), tt.wantErrMsg) {
					t.Errorf("HandleUpdateRoundModalSubmit() error message mismatch: got %q, want substring %q", actualErr.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(actualErr, tt.wantErrIs) {
					t.Errorf("HandleUpdateRoundModalSubmit() error type mismatch: got %T, want %T", actualErr, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleUpdateRoundModalSubmit() unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Errorf("HandleUpdateRoundModalSubmit() unexpected result error: %v", result.Error)
				}
			}
		})
	}
}

func Test_updateRoundManager_HandleUpdateRoundModalCancel(t *testing.T) {
	tests := []struct {
		name         string
		setupSession func(fs *discord.FakeSession)
		args         *discordgo.InteractionCreate
		ctx          context.Context
		wantSuccess  string
		wantErrMsg   string
		wantErrIs    error
	}{
		{
			name: "successful_cancel",
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					if r.Type != discordgo.InteractionResponseChannelMessageWithSource {
						t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", r.Type)
					}
					if r.Data.Content != "Round update cancelled." {
						t.Errorf("Expected content 'Round update cancelled.', got %v", r.Data.Content)
					}
					if r.Data.Flags != discordgo.MessageFlagsEphemeral {
						t.Errorf("Expected ephemeral message, got flags %v", r.Data.Flags)
					}
					return nil
				}
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			},
			ctx:         context.Background(),
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "error sending response",
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					return errors.New("failed to send response")
				}
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID: "interaction-id",
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
				},
			},
			ctx:         context.Background(),
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name:         "nil interaction",
			setupSession: func(fs *discord.FakeSession) {},
			args:         nil,
			ctx:          context.Background(),
			wantSuccess:  "",
			wantErrMsg:   "interaction is nil or incomplete",
			wantErrIs:    nil,
		},
		{
			name: "context cancelled",
			setupSession: func(fs *discord.FakeSession) {
				fs.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantSuccess: "cancelled",
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			fakeInteractionStore := testutils.NewFakeStorage[any]()
			mockLogger := loggerfrolfbot.NoOpLogger

			tt.setupSession(fakeSession)

			urm := &updateRoundManager{
				session:          fakeSession,
				interactionStore: fakeInteractionStore,
				logger:           mockLogger,
				tracer:           noop.NewTracerProvider().Tracer("test"),
				metrics:          &discordmetrics.NoOpMetrics{},
				operationWrapper: testOperationWrapper,
			}
			result, err := urm.HandleUpdateRoundModalCancel(tt.ctx, tt.args)

			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("HandleUpdateRoundModalCancel() CreateRoundOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("HandleUpdateRoundModalCancel() expected error containing %q, got nil", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("HandleUpdateRoundModalCancel() error message mismatch: got %q, want substring %q", err.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("HandleUpdateRoundModalCancel() error type mismatch: got %T, want type %T", err, tt.wantErrIs)
				}
			} else {
				if err != nil {
					t.Errorf("HandleUpdateRoundModalCancel() unexpected error: %v", err)
				}
			}
		})
	}
}
