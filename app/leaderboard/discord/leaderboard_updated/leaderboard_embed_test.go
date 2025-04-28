package leaderboardupdated

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_leaderboardUpdateManager_SendLeaderboardEmbed(t *testing.T) {
	channelID := "test-channel"

	tests := []struct {
		name          string
		setupMocks    func(mockSession *discordmocks.MockSession)
		leaderboard   []LeaderboardEntry
		page          int32
		expectedPage  int32
		expectButtons bool
		expectErr     bool
	}{
		{
			name: "Empty leaderboard",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Verify empty leaderboard message
						if len(send.Embeds) != 1 {
							t.Errorf("Expected 1 embed, got %d", len(send.Embeds))
						}

						embed := send.Embeds[0]
						if embed.Title != "🏆 Leaderboard" {
							t.Errorf("Unexpected title: got %s, want %s", embed.Title, "🏆 Leaderboard")
						}

						if embed.Description != "Page 1/1" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 1/1")
						}

						if len(embed.Fields) != 0 {
							t.Errorf("Expected 0 fields, got %d", len(embed.Fields))
						}

						// No pagination buttons for single page
						if len(send.Components) != 0 {
							t.Errorf("Expected 0 components, got %d", len(send.Components))
						}

						return &discordgo.Message{
							ID:      "test-message-id",
							Embeds:  send.Embeds,
							Content: "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard:   []LeaderboardEntry{},
			page:          1,
			expectedPage:  1,
			expectButtons: false,
			expectErr:     false,
		},
		{
			name: "Single page leaderboard (less than 10 entries)",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Verify single page leaderboard message
						if len(send.Embeds) != 1 {
							t.Errorf("Expected 1 embed, got %d", len(send.Embeds))
						}

						embed := send.Embeds[0]
						if embed.Title != "🏆 Leaderboard" {
							t.Errorf("Unexpected title: got %s, want %s", embed.Title, "🏆 Leaderboard")
						}

						if embed.Description != "Page 1/1" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 1/1")
						}

						if len(embed.Fields) != 5 {
							t.Errorf("Expected 5 fields, got %d", len(embed.Fields))
						}

						// Check first entry
						if embed.Fields[0].Name != "#1" {
							t.Errorf("Unexpected field name: got %s, want %s", embed.Fields[0].Name, "#1")
						}

						if embed.Fields[0].Value != "<@user1>" {
							t.Errorf("Unexpected field value: got %s, want %s", embed.Fields[0].Value, "<@user1>")
						}

						// No pagination buttons for single page
						if len(send.Components) != 0 {
							t.Errorf("Expected 0 components, got %d", len(send.Components))
						}

						return &discordgo.Message{
							ID:      "test-message-id",
							Embeds:  send.Embeds,
							Content: "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard: []LeaderboardEntry{
				{Rank: 1, UserID: "user1"},
				{Rank: 2, UserID: "user2"},
				{Rank: 3, UserID: "user3"},
				{Rank: 4, UserID: "user4"},
				{Rank: 5, UserID: "user5"},
			},
			page:          1,
			expectedPage:  1,
			expectButtons: false,
			expectErr:     false,
		},
		{
			name: "Multi-page leaderboard (first page)",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Verify multi-page leaderboard message (first page)
						if len(send.Embeds) != 1 {
							t.Errorf("Expected 1 embed, got %d", len(send.Embeds))
						}

						embed := send.Embeds[0]
						if embed.Description != "Page 1/2" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 1/2")
						}

						// Should show exactly 10 entries on first page
						if len(embed.Fields) != 10 {
							t.Errorf("Expected 10 fields, got %d", len(embed.Fields))
						}

						// Check pagination buttons
						if len(send.Components) != 1 {
							t.Errorf("Expected 1 component row, got %d", len(send.Components))
							return nil, fmt.Errorf("test failed")
						}

						actionRow, ok := send.Components[0].(discordgo.ActionsRow)
						if !ok {
							t.Errorf("Expected ActionsRow, got %T", send.Components[0])
							return nil, fmt.Errorf("test failed")
						}

						if len(actionRow.Components) != 2 {
							t.Errorf("Expected 2 buttons, got %d", len(actionRow.Components))
							return nil, fmt.Errorf("test failed")
						}

						// Previous button should be disabled, Next enabled
						prevButton, ok := actionRow.Components[0].(discordgo.Button)
						if !ok || !prevButton.Disabled {
							t.Errorf("Expected disabled Previous button")
							return nil, fmt.Errorf("test failed")
						}

						nextButton, ok := actionRow.Components[1].(discordgo.Button)
						if !ok || nextButton.Disabled {
							t.Errorf("Expected enabled Next button")
							return nil, fmt.Errorf("test failed")
						}

						return &discordgo.Message{
							ID:         "test-message-id",
							Embeds:     send.Embeds,
							Components: send.Components,
							Content:    "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard:   createTestLeaderboard(15),
			page:          1,
			expectedPage:  1,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Multi-page leaderboard (second page)",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Verify multi-page leaderboard message (second page)
						embed := send.Embeds[0]
						if embed.Description != "Page 2/2" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 2/2")
						}

						// Should show exactly 5 entries on second page
						if len(embed.Fields) != 5 {
							t.Errorf("Expected 5 fields, got %d", len(embed.Fields))
						}

						// Check first entry on page 2
						if embed.Fields[0].Name != "#11" {
							t.Errorf("Unexpected field name: got %s, want %s", embed.Fields[0].Name, "#11")
						}

						// Previous button should be enabled, Next disabled
						actionRow, ok := send.Components[0].(discordgo.ActionsRow)
						if !ok {
							t.Errorf("Expected ActionsRow, got %T", send.Components[0])
							return nil, fmt.Errorf("test failed")
						}

						prevButton, ok := actionRow.Components[0].(discordgo.Button)
						if !ok || prevButton.Disabled {
							t.Errorf("Expected enabled Previous button")
							return nil, fmt.Errorf("test failed")
						}

						nextButton, ok := actionRow.Components[1].(discordgo.Button)
						if !ok || !nextButton.Disabled {
							t.Errorf("Expected disabled Next button")
							return nil, fmt.Errorf("test failed")
						}

						return &discordgo.Message{
							ID:         "test-message-id",
							Embeds:     send.Embeds,
							Components: send.Components,
							Content:    "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard:   createTestLeaderboard(15),
			page:          2,
			expectedPage:  2,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Page out of range (too low)",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Should correct to page 1
						embed := send.Embeds[0]
						if embed.Description != "Page 1/2" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 1/2")
						}

						return &discordgo.Message{
							ID:         "test-message-id",
							Embeds:     send.Embeds,
							Components: send.Components,
							Content:    "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard:   createTestLeaderboard(15),
			page:          0, // Invalid page, should default to 1
			expectedPage:  1,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Page out of range (too high)",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID string, send *discordgo.MessageSend, options ...any) (*discordgo.Message, error) {
						// Should correct to max page (2)
						embed := send.Embeds[0]
						if embed.Description != "Page 2/2" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 2/2")
						}

						return &discordgo.Message{
							ID:         "test-message-id",
							Embeds:     send.Embeds,
							Components: send.Components,
							Content:    "Test Message",
						}, nil
					}).
					Times(1)
			},
			leaderboard:   createTestLeaderboard(15),
			page:          10, // Invalid page, should default to max (2)
			expectedPage:  2,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Discord API error",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq(channelID), gomock.Any()).
					Return(nil, fmt.Errorf("discord API error")).
					Times(1)
			},
			leaderboard: createTestLeaderboard(5),
			page:        1,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			lum := &leaderboardUpdateManager{
				logger:  mockLogger,
				session: mockSession,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
					return fn(ctx)
				},
			}

			ctx := context.Background()
			got, err := lum.SendLeaderboardEmbed(ctx, channelID, tt.leaderboard, tt.page)

			if (err != nil) != tt.expectErr {
				t.Errorf("SendLeaderboardEmbed() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				// Verify the result contains a discord message
				msg, ok := got.Success.(*discordgo.Message)
				if !ok {
					t.Errorf("SendLeaderboardEmbed() Success is not a *discordgo.Message")
					return
				}

				if len(msg.Embeds) != 1 {
					t.Errorf("SendLeaderboardEmbed() returned message has %d embeds, want 1", len(msg.Embeds))
					return
				}

				// Check if components match expectation
				hasButtons := len(msg.Components) > 0
				if hasButtons != tt.expectButtons {
					t.Errorf("SendLeaderboardEmbed() has buttons = %v, want %v", hasButtons, tt.expectButtons)
				}
			}
		})
	}
}

// Helper function to create test leaderboard entries
func createTestLeaderboard(count int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = LeaderboardEntry{
			Rank:   sharedtypes.TagNumber(i + 1),
			UserID: sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
		}
	}
	return entries
}
