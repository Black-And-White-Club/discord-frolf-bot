package leaderboardupdated

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
	return operationFunc(ctx)
}

// makeTestLeaderboard creates n simple entries for channel-cache seeding.
func makeTestLeaderboard(n int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, n)
	for i := range entries {
		entries[i] = LeaderboardEntry{
			Rank:   sharedtypes.TagNumber(i + 1),
			UserID: sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
		}
	}
	return entries
}

func Test_leaderboardUpdateManager_HandleLeaderboardPagination(t *testing.T) {
	const testChannelID = "test-channel"

	tests := []struct {
		name                string
		customID            string
		cachedLeaderboard   []LeaderboardEntry // pre-seeded cache
		mockRespondErr      error
		expectSuccess       string
		expectFailure       string
		expectErrSubstr     string
		expectRespondCalled bool
	}{
		{
			name:                "pagination successful",
			customID:            "leaderboard_next|2",
			cachedLeaderboard:   makeTestLeaderboard(25), // 3 pages
			expectSuccess:       "pagination updated",
			expectRespondCalled: true,
		},
		{
			name:            "invalid custom_id format",
			customID:        "invalid_format",
			cachedLeaderboard: makeTestLeaderboard(25),
			expectErrSubstr: "invalid CustomID format",
		},
		{
			name:            "invalid page number",
			customID:        "leaderboard_next|invalid",
			cachedLeaderboard: makeTestLeaderboard(25),
			expectErrSubstr: "error parsing page number",
		},
		{
			name:          "no cached data (bot restarted)",
			customID:      "leaderboard_next|2",
			// cachedLeaderboard intentionally empty / nil
			expectFailure:       "no cached leaderboard data",
			expectRespondCalled: true, // sends an ephemeral cache-miss message
		},
		{
			name:                "page clamped to min (page 0 → page 1)",
			customID:            "leaderboard_prev|0",
			cachedLeaderboard:   makeTestLeaderboard(25),
			expectSuccess:       "pagination updated",
			expectRespondCalled: true,
		},
		{
			name:                "page clamped to max (page 999 → last page)",
			customID:            "leaderboard_next|999",
			cachedLeaderboard:   makeTestLeaderboard(25),
			expectSuccess:       "pagination updated",
			expectRespondCalled: true,
		},
		{
			name:                "discord respond error",
			customID:            "leaderboard_next|2",
			cachedLeaderboard:   makeTestLeaderboard(25),
			mockRespondErr:      errors.New("discord API error"),
			expectErrSubstr:     "error updating leaderboard message",
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
				session:          fakeSession,
				logger:           slog.Default(),
				operationWrapper: testOperationWrapper,
				dataByChannelID:  make(map[string][]LeaderboardEntry),
				messageByChannelID: make(map[string]string),
			}

			// Seed the cache when the test requires it
			if len(tt.cachedLeaderboard) > 0 {
				manager.setCachedLeaderboard(testChannelID, tt.cachedLeaderboard)
			}

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:        "test-interaction",
					ChannelID: testChannelID,
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

			result, err := manager.HandleLeaderboardPagination(context.Background(), interaction)

			// Error substring check (lives in result.Error for non-API errors)
			if tt.expectErrSubstr != "" {
				var target error
				if result.Error != nil {
					target = result.Error
				} else if err != nil {
					target = err
				}
				if target == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectErrSubstr)
				} else if !strings.Contains(target.Error(), tt.expectErrSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectErrSubstr, target.Error())
				}
			}

			// API-level error propagation
			if tt.mockRespondErr != nil {
				if err == nil {
					t.Error("expected error from InteractionRespond, got nil")
				}
			} else if tt.expectErrSubstr == "" {
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

			if tt.expectRespondCalled && !respondCalled {
				t.Error("expected InteractionRespond to be called, but it was not")
			}
		})
	}
}
