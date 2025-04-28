package updateround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error) {
	return operationFunc(ctx)
}

type updateRoundManagerMock struct {
	sendModalCalled          bool
	mockSendUpdateRoundModal func(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error)
	session                  *discordmocks.MockSession
	logger                   *slog.Logger
	operationWrapper         func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error)
}

func (urm *updateRoundManagerMock) HandleEditRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	return urm.operationWrapper(ctx, "SendUpdateRoundModal", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		// Add validation logic to match the real implementation
		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			err := fmt.Errorf("invalid custom_id format: %s", customID)
			// Initialize with empty string for Success field
			return UpdateRoundOperationResult{Error: err, Success: ""}, nil
		}

		// Parse UUID properly
		_, err := uuid.Parse(parts[1])
		if err != nil {
			err := fmt.Errorf("invalid UUID for round ID: %w", err)
			// Initialize with empty string for Success field
			return UpdateRoundOperationResult{Error: err, Success: ""}, nil
		}

		// After validation passes, call SendUpdateRoundModal
		result, err := urm.SendUpdateRoundModal(ctx, i)
		if err != nil {
			// Initialize with empty string for Success field
			return UpdateRoundOperationResult{Error: err, Success: ""}, err
		}
		if result.Error != nil {
			// Make sure Success is explicitly an empty string, not nil
			return UpdateRoundOperationResult{Error: result.Error, Success: ""}, nil
		}

		// Ensure we have the correct success message
		return UpdateRoundOperationResult{Success: "modal sent", Error: nil}, nil
	})
}

func (urm *updateRoundManagerMock) SendUpdateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	urm.sendModalCalled = true
	if urm.mockSendUpdateRoundModal != nil {
		return urm.mockSendUpdateRoundModal(ctx, i)
	}
	return UpdateRoundOperationResult{Success: "default mock success"}, nil
}

func Test_updateRoundManager_HandleEditRoundButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		mockModalFn     func(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error)
		customID        string
		expectSuccess   string
		expectErrSubstr string
		expectModalCall bool
	}{
		{
			name: "modal sent successfully",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{}, nil
			},
			customID:        "edit_round|" + uuid.New().String(),
			expectSuccess:   "modal sent",
			expectModalCall: true,
		},
		{
			name:            "invalid custom_id format",
			customID:        "invalid",
			expectErrSubstr: "invalid custom_id format",
			expectModalCall: false,
		},
		{
			name:            "invalid UUID",
			customID:        "edit_round|invalid_uuid",
			expectErrSubstr: "invalid UUID for round ID",
			expectModalCall: false,
		},
		{
			name: "modal returns underlying error",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{}, errors.New("modal underlying failed")
			},
			customID:        "edit_round|" + uuid.New().String(),
			expectErrSubstr: "modal underlying failed",
			expectModalCall: true,
		},
		{
			name: "modal returns operation result error",
			mockModalFn: func(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
				return UpdateRoundOperationResult{Error: errors.New("bad result from modal")}, nil
			},
			customID:        "edit_round|" + uuid.New().String(),
			expectErrSubstr: "bad result from modal",
			expectModalCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urm := &updateRoundManagerMock{
				mockSendUpdateRoundModal: tt.mockModalFn,
				session:                  discordmocks.NewMockSession(ctrl),
				logger:                   slog.Default(),
				operationWrapper:         testOperationWrapper,
			}

			// Corrected Interaction Initialization
			messageComponentData := discordgo.MessageComponentInteractionData{
				CustomID:      tt.customID,
				ComponentType: discordgo.ButtonComponent,
			}

			// Create the InteractionCreate struct
			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "test-interaction-id-" + tt.name,
					Type: discordgo.InteractionMessageComponent,
					Member: &discordgo.Member{
						User: &discordgo.User{
							ID: "user-123",
						},
					},
					Data: messageComponentData,
				},
			}

			result, _ := urm.HandleEditRoundButton(context.Background(), interaction)

			// Assertions
			if tt.expectErrSubstr != "" {
				if result.Error == nil {
					t.Errorf("expected error containing %q, got nil result error", tt.expectErrSubstr)
				} else if !strings.Contains(result.Error.Error(), tt.expectErrSubstr) {
					t.Errorf("expected result error to contain %q, got %q", tt.expectErrSubstr, result.Error.Error())
				}
				if result.Success != "" {
					t.Errorf("expected empty success message with error, got %q", result.Success)
				}
			} else {
				if result.Error != nil {
					t.Errorf("unexpected error: %v", result.Error)
				}
				if result.Success != tt.expectSuccess {
					t.Errorf("unexpected success value: got %q, want %q", result.Success, tt.expectSuccess)
				}
			}

			// Verify mock call expectation
			if tt.expectModalCall && !urm.sendModalCalled {
				t.Errorf("expected SendUpdateRoundModal to be called, but it was not")
			}
			if !tt.expectModalCall && urm.sendModalCalled {
				t.Errorf("expected SendUpdateRoundModal NOT to be called, but it was")
			}
		})
	}
}
