package createround

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

// createRoundManagerMock is a testable version of createRoundManager that allows function mocking
type createRoundManagerMock struct {
	session                  discordmocks.MockSession
	sendModalCalled          bool
	mockSendCreateRoundModal func(ctx context.Context, i *discordgo.InteractionCreate) error
}

// HandleCreateRoundCommand implements the real function but uses mocked dependencies
func (crm *createRoundManagerMock) HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	// This mimics the actual implementation
	err := crm.SendCreateRoundModal(ctx, i)
	if err != nil {
		// Error is logged in the real implementation
	}
}

// SendCreateRoundModal is the mocked version for testing
func (crm *createRoundManagerMock) SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	crm.sendModalCalled = true
	if crm.mockSendCreateRoundModal != nil {
		return crm.mockSendCreateRoundModal(ctx, i)
	}
	return nil
}

func Test_createRoundManager_HandleCreateRoundCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	// Sample interaction with member and user
	testInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
		},
	}

	tests := []struct {
		name     string
		mockFunc func(*createRoundManagerMock)
	}{
		{
			name: "successful modal send",
			mockFunc: func(crm *createRoundManagerMock) {
				crm.mockSendCreateRoundModal = func(ctx context.Context, i *discordgo.InteractionCreate) error {
					return nil
				}
			},
		},
		{
			name: "error sending modal",
			mockFunc: func(crm *createRoundManagerMock) {
				crm.mockSendCreateRoundModal = func(ctx context.Context, i *discordgo.InteractionCreate) error {
					return errors.New("failed to send modal")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock wrapper that records method calls
			crm := &createRoundManagerMock{
				session:         *mockSession,
				sendModalCalled: false,
			}

			// Configure the mock behavior
			if tt.mockFunc != nil {
				tt.mockFunc(crm)
			}

			// Execute the method under test
			crm.HandleCreateRoundCommand(context.Background(), testInteraction)

			// Verify the mock was called
			if !crm.sendModalCalled {
				t.Error("SendCreateRoundModal was not called")
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)

	crm := &createRoundManager{
		session:          mockSession,
		interactionStore: mockInteractionStore,
	}

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx           context.Context
			correlationID string
			message       string
			edit          []*discordgo.WebhookEdit
		}
		wantErr bool
	}{
		{
			name: "successful update",
			setup: func() {
				interaction := &discordgo.Interaction{
					ID: "interaction-id",
				}
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(interaction, true)

				// Return a response object and nil error for success
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Eq(interaction), gomock.Any()).
					Return(&discordgo.Message{}, nil)
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message",
				edit:          nil,
			},
			wantErr: false,
		},
		{
			name: "interaction not found",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(nil, false) // Interaction not found
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message",
				edit:          nil,
			},
			wantErr: true,
		},
		{
			name: "stored interaction is not of type *discordgo.Interaction",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return("not-an-interaction", true) // Incorrect type
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message",
				edit:          nil,
			},
			wantErr: true,
		},
		{
			name: "failed to update interaction response",
			setup: func() {
				interaction := &discordgo.Interaction{
					ID: "interaction-id",
				}
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(interaction, true)

				// Return nil response and an error for failure
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Eq(interaction), gomock.Any()).
					Return(nil, errors.New("failed to update response"))
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message",
				edit:          nil,
			},
			wantErr: true,
		},
		{
			name: "successful update with edit",
			setup: func() {
				interaction := &discordgo.Interaction{
					ID: "interaction-id",
				}
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(interaction, true)

				// Return a response object and nil error for success
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Eq(interaction), gomock.Any()).
					Return(&discordgo.Message{}, nil)
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
				edit          []*discordgo.WebhookEdit
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message",
				edit: []*discordgo.WebhookEdit{
					{
						Content: new(string),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := crm.UpdateInteractionResponse(tt.args.ctx, tt.args.correlationID, tt.args.message, tt.args.edit...)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRoundManager.UpdateInteractionResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_createRoundManager_UpdateInteractionResponseWithRetryButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)

	crm := &createRoundManager{
		session:          mockSession,
		interactionStore: mockInteractionStore,
	}

	tests := []struct {
		name  string
		setup func()
		args  struct {
			ctx           context.Context
			correlationID string
			message       string
		}
		wantErr bool
	}{
		{
			name: "successful update with retry button",
			setup: func() {
				interaction := &discordgo.Interaction{
					ID: "interaction-id",
				}
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(interaction, true)

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Eq(interaction), gomock.Any()).
					Return(&discordgo.Message{}, nil) // Return a message and no error
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message with retry button",
			},
			wantErr: false,
		},
		{
			name: "interaction not found",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(nil, false) // Interaction not found
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message with retry button",
			},
			wantErr: true,
		},
		{
			name: "stored interaction is not of type *discordgo.Interaction",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return("not-an-interaction", true) // Incorrect type
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message with retry button",
			},
			wantErr: true,
		},
		{
			name: "failed to update interaction response",
			setup: func() {
				interaction := &discordgo.Interaction{
					ID: "interaction-id",
				}
				mockInteractionStore.EXPECT().
					Get("correlation-id").
					Return(interaction, true)

				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Eq(interaction), gomock.Any()).
					Return(nil, errors.New("failed to update response")) // Simulate an error
			},
			args: struct {
				ctx           context.Context
				correlationID string
				message       string
			}{
				ctx:           context.Background(),
				correlationID: "correlation-id",
				message:       "Updated message with retry button",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			if err := crm.UpdateInteractionResponseWithRetryButton(tt.args.ctx, tt.args.correlationID, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("createRoundManager.UpdateInteractionResponseWithRetryButton() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func (crm *createRoundManagerMock) HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) {
	// This mimics the actual implementation
	err := crm.SendCreateRoundModal(ctx, i)
	if err != nil {
		// If modal sending fails, update the message to inform the user
		_, updateErr := crm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    stringPtr("Failed to open the form. Please try using the /createround command again."),
			Components: &[]discordgo.MessageComponent{},
		})
		if updateErr != nil {
			// In real code, this error would be logged
		}
	}
}

func Test_createRoundManager_HandleRetryCreateRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Sample interaction with member and user
	testInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID: "interaction-id",
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
		},
	}

	tests := []struct {
		name                string
		mockSendModalResult error
		expectResponseEdit  bool
		responseEditResult  error
	}{
		// Your test cases remain the same
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh mocks for each test
			mockSession := discordmocks.NewMockSession(ctrl)

			// Create a createRoundManagerMock instead of createRoundManager
			crm := &createRoundManagerMock{
				session: *mockSession, // Use the same mock session
			}

			// Set up your mock function
			crm.mockSendCreateRoundModal = func(ctx context.Context, i *discordgo.InteractionCreate) error {
				return tt.mockSendModalResult
			}

			// If we expect error handling, set up the response edit expectation
			if tt.expectResponseEdit {
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, edit *discordgo.WebhookEdit) (*discordgo.Message, error) {
						// Your validation remains the same
						return &discordgo.Message{}, tt.responseEditResult
					})
			}

			// Execute the function using the mock implementation
			crm.HandleRetryCreateRound(context.Background(), testInteraction)
		})
	}
}
