package leaderboardupdated

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
	return operationFunc(ctx)
}

func Test_leaderboardUpdateManager_HandleLeaderboardPagination(t *testing.T) {
	tests := []struct {
		name                string
		customID            string
		mockRespondErr      error
		expectFailure       string
		expectErrSubstr     string
		expectRespondCalled bool
	}{
		{
			name:                "old pagination button handled gracefully",
			customID:            "leaderboard_next|2",
			expectFailure:       "pagination no longer supported",
			expectRespondCalled: true,
		},
		{
			name:            "invalid custom_id format",
			customID:        "invalid_format",
			expectErrSubstr: "invalid CustomID format",
		},
		{
			name:            "invalid page number",
			customID:        "leaderboard_next|invalid",
			expectErrSubstr: "error parsing page number",
		},
		{
			name:                "discord respond error on stale button",
			customID:            "leaderboard_next|2",
			mockRespondErr:      errors.New("discord API error"),
			expectFailure:       "pagination no longer supported", // still returns failure, not error
			expectRespondCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()

			respondCalled := false
			fakeSession.InteractionRespondFunc = func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
				respondCalled = true
				return tt.mockRespondErr
			}

			manager := &leaderboardUpdateManager{
				session:            fakeSession,
				logger:             slog.Default(),
				operationWrapper:   testOperationWrapper,
				messageByChannelID: make(map[string]string),
			}

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:        "test-interaction",
					ChannelID: "test-channel",
					Type:      discordgo.InteractionMessageComponent,
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user123"},
					},
					Data: discordgo.MessageComponentInteractionData{
						CustomID:      tt.customID,
						ComponentType: discordgo.ButtonComponent,
					},
					Message: &discordgo.Message{},
				},
			}

			result, _ := manager.HandleLeaderboardPagination(context.Background(), interaction)

			if tt.expectErrSubstr != "" {
				if result.Error == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectErrSubstr)
				} else if !strings.Contains(result.Error.Error(), tt.expectErrSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectErrSubstr, result.Error.Error())
				}
			}

			if tt.expectFailure != "" {
				if result.Failure != tt.expectFailure {
					t.Errorf("expected failure %q, got %q", tt.expectFailure, result.Failure)
				}
			}

			if tt.expectRespondCalled && !respondCalled {
				t.Error("expected InteractionRespond to be called, but it was not")
			}
		})
	}
}
