package leaderboardupdated

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
	return operationFunc(ctx)
}

func Test_leaderboardUpdateManager_HandleLeaderboardPagination(t *testing.T) {
	tests := []struct {
		name                string
		customID            string
		embedDescription    string
		embedFields         []*discordgo.MessageEmbedField
		mockRespondErr      error
		expectSuccess       string
		expectFailure       string
		expectErrSubstr     string
		expectRespondCalled bool
	}{
		{
			name:                "pagination successful",
			customID:            "leaderboard_next|2",
			embedDescription:    "Page 1/3",
			embedFields:         make([]*discordgo.MessageEmbedField, 25), // Multiple pages of data
			expectSuccess:       "pagination updated",
			expectRespondCalled: true,
		},
		{
			name:             "invalid custom_id format",
			customID:         "invalid_format",
			embedDescription: "Page 1/3",
			embedFields:      make([]*discordgo.MessageEmbedField, 25),
			expectErrSubstr:  "invalid CustomID format",
		},
		{
			name:             "invalid page number",
			customID:         "leaderboard_next|invalid",
			embedDescription: "Page 1/3",
			embedFields:      make([]*discordgo.MessageEmbedField, 25),
			expectErrSubstr:  "error parsing page number",
		},
		{
			name:            "no embeds in message",
			customID:        "leaderboard_next|2",
			expectErrSubstr: "no embeds found in message",
		},
		{
			name:             "invalid embed description format",
			customID:         "leaderboard_next|2",
			embedDescription: "Invalid description",
			embedFields:      make([]*discordgo.MessageEmbedField, 25),
			expectErrSubstr:  "error parsing embed page numbers",
		},
		{
			name:             "page out of bounds - below min",
			customID:         "leaderboard_prev|0",
			embedDescription: "Page 1/3",
			embedFields:      make([]*discordgo.MessageEmbedField, 25),
			expectFailure:    "page out of range",
		},
		{
			name:             "page out of bounds - above max",
			customID:         "leaderboard_next|999",
			embedDescription: "Page 1/3",
			embedFields:      make([]*discordgo.MessageEmbedField, 25),
			expectFailure:    "page out of range",
		},
		{
			name:                "discord respond error",
			customID:            "leaderboard_next|2",
			embedDescription:    "Page 1/3",
			embedFields:         make([]*discordgo.MessageEmbedField, 25),
			mockRespondErr:      errors.New("discord API error"),
			expectErrSubstr:     "error updating leaderboard message",
			expectRespondCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock session
			mockSession := discordmocks.NewMockSession(ctrl)

			// Setup mock for InteractionRespond if needed
			if tt.expectRespondCalled {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(tt.mockRespondErr).
					Times(1)
			}

			// Create the real leaderboard manager with mocked dependencies
			manager := &leaderboardUpdateManager{
				session:          mockSession,
				logger:           slog.Default(),
				operationWrapper: testOperationWrapper, // Use a simple wrapper for testing
			}

			// Create interaction with proper structure
			var embeds []*discordgo.MessageEmbed
			if tt.embedDescription != "" {
				embeds = []*discordgo.MessageEmbed{
					{
						Description: tt.embedDescription,
						Fields:      tt.embedFields,
					},
				}
			}

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "test-interaction",
					Type: discordgo.InteractionMessageComponent,
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user123"},
					},
					Data: discordgo.MessageComponentInteractionData{
						CustomID:      tt.customID,
						ComponentType: discordgo.ButtonComponent,
					},
					Message: &discordgo.Message{
						Embeds: embeds,
					},
				},
			}

			// Call the method with context and capture both return values
			result, err := manager.HandleLeaderboardPagination(context.Background(), interaction)

			// Modify the assertion section to check result.Error instead of err
			if tt.expectErrSubstr != "" {
				if result.Error == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectErrSubstr)
				} else if !strings.Contains(result.Error.Error(), tt.expectErrSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectErrSubstr, result.Error.Error())
				}
			}

			// Remove the err checks for non-API errors
			if tt.mockRespondErr != nil {
				if err == nil {
					t.Error("expected error from InteractionRespond, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.expectFailure != "" {
				if result.Failure != tt.expectFailure {
					t.Errorf("expected failure %q, got %q", tt.expectFailure, result.Failure)
				}
			}

			if tt.expectSuccess != "" {
				if result.Success != tt.expectSuccess {
					t.Errorf("expected success %q, got %q", tt.expectSuccess, result.Success)
				}
			}
		})
	}
}
