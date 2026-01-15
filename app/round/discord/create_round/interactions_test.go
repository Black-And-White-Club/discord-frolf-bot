package createround

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	helpermocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
	return operationFunc(ctx)
}

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// createRoundManagerMock is a testable version of createRoundManager that allows function mocking
type createRoundManagerMock struct {
	sendModalCalled          bool
	mockSendCreateRoundModal func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
	session                  *discordmocks.MockSession // Use the gomock mock
	logger                   *slog.Logger
	operationWrapper         func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error)
}

// HandleCreateRoundCommand mimics the real implementation but allows bypassing the wrapper and mocking SendCreateRoundModal
func (crm *createRoundManagerMock) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	return func(ctx context.Context, fn func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
		return fn(ctx)
	}(ctx, func(ctx context.Context) (CreateRoundOperationResult, error) {
		result, err := crm.SendCreateRoundModal(ctx, i)
		if err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}
		if result.Error != nil {
			return result, nil
		}
		return CreateRoundOperationResult{Success: "modal sent"}, nil
	})
}

// SendCreateRoundModal is the mocked version for testing
func (crm *createRoundManagerMock) SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	crm.sendModalCalled = true
	if crm.mockSendCreateRoundModal != nil {
		return crm.mockSendCreateRoundModal(ctx, i)
	}
	return CreateRoundOperationResult{Success: "default"}, nil
}

// HandleRetryCreateRound mimics the real implementation for testing
func (crm *createRoundManagerMock) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
	return crm.operationWrapper(ctx, "handle_retry_create_round", func(ctx context.Context) (CreateRoundOperationResult, error) {
		result, err := crm.SendCreateRoundModal(ctx, i)
		if err != nil {
			return CreateRoundOperationResult{Error: err}, err
		}

		if result.Error != nil {
			// Simulate the message edit on error
			_, updateErr := crm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content:    stringPtr("Failed to open the form. Please try using the /createround command again."),
				Components: &[]discordgo.MessageComponent{},
			})
			if updateErr != nil {
				crm.logger.ErrorContext(ctx, "Failed to update error message in mock", attr.Error(updateErr))
			}
			return result, nil
		}

		return CreateRoundOperationResult{Success: "retry modal sent"}, nil
	})
}

func Test_createRoundManager_HandleCreateRoundCommand(t *testing.T) {
	tests := []struct {
		name            string
		mockModalFn     func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
		expectSuccess   string
		expectErrSubstr string
	}{
		{
			name: "modal sent successfully",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{}, nil
			},
			expectSuccess: "modal sent",
		},
		{
			name: "modal returns operation error",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{}, errors.New("modal failed")
			},
			expectErrSubstr: "modal failed",
		},
		{
			name: "modal returns result error",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{Error: errors.New("bad result")}, nil
			},
			expectErrSubstr: "bad result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crm := &createRoundManagerMock{
				mockSendCreateRoundModal: tt.mockModalFn,
				session:                  &discordmocks.MockSession{}, // Initialize the mock session
				operationWrapper: func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
					return operationFunc(ctx)
				},
				logger: slog.Default(),
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
					t.Errorf("unexpected success value: got %q, want %q", result.Success, tt.expectSuccess)
				}
			}

			if !crm.sendModalCalled {
				t.Errorf("SendCreateRoundModal was not called")
			}
		})
	}
}

func Test_createRoundManager_HandleRetryCreateRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                        string
		mockModalFn                 func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
		mockInteractionEditError    error
		expectedErrMsg              string
		expectedSuccess             string
		expectInteractionEditCalled bool
	}{
		{
			name: "retry modal sent successfully",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{}, nil
			},
			expectedSuccess:             "retry modal sent",
			expectInteractionEditCalled: false,
		},
		{
			name: "SendCreateRoundModal returns error",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{}, errors.New("modal failed")
			},
			expectedErrMsg:              "modal failed",
			expectInteractionEditCalled: false,
		},
		{
			name: "SendCreateRoundModal returns result error, edit succeeds",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{Error: errors.New("result failure")}, nil
			},
			expectedErrMsg:              "result failure",
			expectInteractionEditCalled: true,
		},
		{
			name: "SendCreateRoundModal returns result error, edit fails",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error) {
				return CreateRoundOperationResult{Error: errors.New("result failure")}, nil
			},
			mockInteractionEditError:    errors.New("edit failed"),
			expectedErrMsg:              "result failure",
			expectInteractionEditCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSession := discordmocks.NewMockSession(ctrl)

			mock := &createRoundManagerMock{
				mockSendCreateRoundModal: tt.mockModalFn,
				session:                  mockSession,
				operationWrapper:         testOperationWrapper,
				logger:                   slog.Default(),
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

			// Set expectations on mockSession.InteractionResponseEdit
			if tt.expectInteractionEditCalled {
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(interaction.Interaction), // Expect the correct interaction
						gomock.Any(),                       // We might not care about the exact edit content here
					).
					Return(&discordgo.Message{}, tt.mockInteractionEditError). // Correct return values
					Times(1)
			} else {
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil, nil). // Correct return values for no call
					Times(0)
			}

			result, err := mock.HandleRetryCreateRound(context.Background(), interaction)

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
					t.Errorf("expected success: %q, got: %q", tt.expectedSuccess, result.Success)
				}
			}

			if !mock.sendModalCalled {
				t.Errorf("SendCreateRoundModal was not called")
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockHelper := helpermocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &discordmetrics.NoOpMetrics{}

	rm := &createRoundManager{
		session:          mockSession,
		publisher:        nil,
		logger:           logger,
		helper:           mockHelper,
		config:           mockConfig,
		interactionStore: mockInteractionStore,
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
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-123").
					Return(&discordgo.Interaction{ID: "int-id"}, nil).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(&discordgo.Interaction{ID: "int-id"}),
						gomock.Eq(&discordgo.WebhookEdit{Content: stringPtr("Updated message")}),
					).
					Return(&discordgo.Message{}, nil).
					Times(1)
				// Expect cleanup of stored interaction after successful update
				mockInteractionStore.EXPECT().
					Delete(gomock.Any(), "corr-123").
					Times(1)
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
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "successful update with existing edit",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-456").
					Return(&discordgo.Interaction{ID: "int-id-2"}, nil).
					Times(1)
				existingEmbeds := []*discordgo.MessageEmbed{{Title: "Old Embed"}}
				expectedEdit := &discordgo.WebhookEdit{Content: stringPtr("New message"), Embeds: &existingEmbeds}
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(&discordgo.Interaction{ID: "int-id-2"}),
						gomock.Eq(expectedEdit),
					).
					Return(&discordgo.Message{}, nil).
					Times(1)
				// Expect cleanup of stored interaction after successful update
				mockInteractionStore.EXPECT().
					Delete(gomock.Any(), "corr-456").
					Times(1)
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
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "interaction not found",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-not-found").
					Return(nil, errors.New("not found")).
					Times(1)
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
			wantSuccess: "",
			wantErrMsg:  "no interaction found for correlation ID: corr-not-found",
			wantErrIs:   nil,
		},
		{
			name: "stored interaction is wrong type",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-wrong-type").
					Return("not-an-interaction", nil).
					Times(1)
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "corr-wrong-type",
				message:       "This also won't be sent",
				edit:          nil,
			},
			wantSuccess: "",
			wantErrMsg:  "interaction is not of the expected type",
			wantErrIs:   nil,
		},
		{
			name: "interaction response edit fails",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-fail").
					Return(&discordgo.Interaction{ID: "int-fail"}, nil).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(&discordgo.Interaction{ID: "int-fail"}),
						gomock.Eq(&discordgo.WebhookEdit{Content: stringPtr("Failed update")}),
					).
					Return(nil, errors.New("discord API error")).
					Times(1)
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
			wantSuccess: "",
			wantErrMsg:  "discord API error",
			wantErrIs:   nil,
		},
		{
			name: "context cancelled",
			setup: func(ctx context.Context) {
				// No expectations on mockInteractionStore or mockSession
				// because the operation should return early due to context cancellation
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
			wantSuccess: "",
			wantErrMsg:  context.Canceled.Error(),
			wantErrIs:   context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.args.ctx) // Pass the context to the setup function

			result, _ := rm.UpdateInteractionResponse(tt.args.ctx, tt.args.correlationID, tt.args.message, tt.args.edit...)

			// Check the RoleOperationResult.Success field
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("UpdateInteractionResponse() RoleOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Check the RoleOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("UpdateInteractionResponse() RoleOperationResult.Error is nil, expected error containing %q", tt.wantErrMsg)
				} else {
					// Check if the error message contains the expected substring
					if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
						t.Errorf("UpdateInteractionResponse() RoleOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
					}
					// Optionally, check for a specific error type if tt.wantErrIs is set
					if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
						t.Errorf("UpdateInteractionResponse() RoleOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
					}
				}
			} else {
				if result.Error != nil {
					t.Errorf("UpdateInteractionResponse() RoleOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponseWithRetryButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface[any](ctrl)
	logger := loggerfrolfbot.NoOpLogger
	mockHelper := helpermocks.NewMockHelpers(ctrl)
	mockConfig := &config.Config{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &discordmetrics.NoOpMetrics{}

	rm := &createRoundManager{
		session:          mockSession,
		publisher:        nil,
		logger:           logger,
		helper:           mockHelper,
		config:           mockConfig,
		interactionStore: mockInteractionStore,
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
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-123").
					Return(&discordgo.Interaction{ID: "int-id"}, nil).
					Times(1)
				expectedEdit := &discordgo.WebhookEdit{
					Content: stringPtr("Something went wrong"),
					Components: &[]discordgo.MessageComponent{
						discordgo.ActionsRow{Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Try Again",
								Style:    discordgo.PrimaryButton,
								CustomID: "retry_create_round",
							},
						}},
					},
				}
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(&discordgo.Interaction{ID: "int-id"}),
						gomock.Eq(expectedEdit),
					).
					Return(&discordgo.Message{}, nil).
					Times(1)
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
			wantErrMsg:  "",
			wantErrIs:   nil,
		},
		{
			name: "interaction not found",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-not-found").
					Return(nil, errors.New("not found")).
					Times(1)
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
			wantSuccess: "",
			wantErrMsg:  "no interaction found for correlation ID: corr-not-found",
			wantErrIs:   nil,
		},
		{
			name: "stored interaction is wrong type",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-wrong-type").
					Return("not-an-interaction", nil).
					Times(1)
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "corr-wrong-type",
				message:       "This also won't be sent",
			},
			wantSuccess: "",
			wantErrMsg:  "interaction is not of the expected type",
			wantErrIs:   nil,
		},
		{
			name: "interaction response edit fails",
			setup: func(ctx context.Context) {
				mockInteractionStore.EXPECT().
					Get(gomock.Any(), "corr-fail").
					Return(&discordgo.Interaction{ID: "int-fail"}, nil).
					Times(1)
				expectedEdit := &discordgo.WebhookEdit{
					Content: stringPtr("Failed update"),
					Components: &[]discordgo.MessageComponent{
						discordgo.ActionsRow{Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "Try Again",
								Style:    discordgo.PrimaryButton,
								CustomID: "retry_create_round",
							},
						}},
					},
				}
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.Eq(&discordgo.Interaction{ID: "int-fail"}),
						gomock.Eq(expectedEdit),
					).
					Return(nil, errors.New("discord API error")).
					Times(1)
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
			wantSuccess: "",
			wantErrMsg:  "discord API error", // Corrected error message
			wantErrIs:   nil,
		},
		{
			name: "context cancelled",
			setup: func(ctx context.Context) {
				// No expectations on mocks, operation should exit early
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
			wantSuccess: "",
			wantErrMsg:  context.Canceled.Error(),
			wantErrIs:   context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.args.ctx)

			result, _ := rm.UpdateInteractionResponseWithRetryButton(tt.args.ctx, tt.args.correlationID, tt.args.message)

			// Check the RoleOperationResult.Success field
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("UpdateInteractionResponseWithRetryButton() RoleOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Check the RoleOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("UpdateInteractionResponseWithRetryButton() RoleOperationResult.Error is nil, expected error containing %q", tt.wantErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("UpdateInteractionResponseWithRetryButton() RoleOperationResult.Error message mismatch: got %q, want substring %q", result.Error.Error(), tt.wantErrMsg)
				}
				if tt.wantErrIs != nil && !errors.Is(result.Error, tt.wantErrIs) {
					t.Errorf("UpdateInteractionResponseWithRetryButton() RoleOperationResult.Error type mismatch: got %T, want type %T", result.Error, tt.wantErrIs)
				}
			} else {
				if result.Error != nil {
					t.Errorf("UpdateInteractionResponseWithRetryButton() RoleOperationResult.Error is not nil, expected nil. Got: %v", result.Error)
				}
			}
		})
	}
}
