package leaderboardupdated

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func Test_leaderboardUpdateManager_SendLeaderboardEmbed(t *testing.T) {
	channelID := "test-channel"

	tests := []struct {
		name          string
		setupFake     func(t *testing.T, fakeSession *discord.FakeSession)
		leaderboard   []LeaderboardEntry
		page          int32
		expectedPage  int32
		expectButtons bool
		expectErr     bool
	}{
		{
			name: "Empty leaderboard",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
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
				}
			},
			leaderboard:   []LeaderboardEntry{},
			page:          1,
			expectedPage:  1,
			expectButtons: false,
			expectErr:     false,
		},
		{
			name: "Single page leaderboard (less than 10 entries)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
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

					if len(embed.Fields) != 1 {
						t.Errorf("Expected 1 field, got %d", len(embed.Fields))
					}

					// Check the single Tags field
					if embed.Fields[0].Name != "Tags" {
						t.Errorf("Unexpected field name: got %s, want %s", embed.Fields[0].Name, "Tags")
					}

					expectedValue := "🥇 **Tag #1  ** <@user1>\n🥈 **Tag #2  ** <@user2>\n🥉 **Tag #3  ** <@user3>\n🏷️ **Tag #4  ** <@user4>\n🗑️ **Tag #5  ** <@user5>\n"
					if embed.Fields[0].Value != expectedValue {
						t.Errorf("Unexpected field value: got %s, want %s", embed.Fields[0].Value, expectedValue)
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
				}
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
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					// Verify multi-page leaderboard message (first page)
					if len(send.Embeds) != 1 {
						t.Errorf("Expected 1 embed, got %d", len(send.Embeds))
					}

					embed := send.Embeds[0]
					if embed.Description != "Page 1/2" {
						t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 1/2")
					}

					// Should show exactly 1 field with 10 entries on first page
					if len(embed.Fields) != 1 {
						t.Errorf("Expected 1 field, got %d", len(embed.Fields))
					}

					// Check the Tags field contains all 10 entries
					if embed.Fields[0].Name != "Tags" {
						t.Errorf("Unexpected field name: got %s, want %s", embed.Fields[0].Name, "Tags")
					}

					// Should contain entries 1-10 with proper emojis
					expectedValue := "🥇 **Tag #1  ** <@user1>\n🥈 **Tag #2  ** <@user2>\n🥉 **Tag #3  ** <@user3>\n🏷️ **Tag #4  ** <@user4>\n🏷️ **Tag #5  ** <@user5>\n🏷️ **Tag #6  ** <@user6>\n🏷️ **Tag #7  ** <@user7>\n🏷️ **Tag #8  ** <@user8>\n🏷️ **Tag #9  ** <@user9>\n🏷️ **Tag #10 ** <@user10>\n"
					if embed.Fields[0].Value != expectedValue {
						t.Errorf("Unexpected field value: got %s, want %s", embed.Fields[0].Value, expectedValue)
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
				}
			},
			leaderboard:   createTestLeaderboard(15),
			page:          1,
			expectedPage:  1,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Multi-page leaderboard (second page)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					// Verify multi-page leaderboard message (second page)
					embed := send.Embeds[0]
					if embed.Description != "Page 2/2" {
						t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Page 2/2")
					}

					// Should show exactly 1 field with 5 entries on second page
					if len(embed.Fields) != 1 {
						t.Errorf("Expected 1 field, got %d", len(embed.Fields))
					}

					// Check the Tags field contains entries 11-15
					if embed.Fields[0].Name != "Tags" {
						t.Errorf("Unexpected field name: got %s, want %s", embed.Fields[0].Name, "Tags")
					}

					// Should contain entries 11-15 with last place emoji for #15
					expectedValue := "🏷️ **Tag #11 ** <@user11>\n🏷️ **Tag #12 ** <@user12>\n🏷️ **Tag #13 ** <@user13>\n🏷️ **Tag #14 ** <@user14>\n🗑️ **Tag #15 ** <@user15>\n"
					if embed.Fields[0].Value != expectedValue {
						t.Errorf("Unexpected field value: got %s, want %s", embed.Fields[0].Value, expectedValue)
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
				}
			},
			leaderboard:   createTestLeaderboard(15),
			page:          2,
			expectedPage:  2,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Page out of range (too low)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
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
				}
			},
			leaderboard:   createTestLeaderboard(15),
			page:          0, // Invalid page, should default to 1
			expectedPage:  1,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Page out of range (too high)",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
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
				}
			},
			leaderboard:   createTestLeaderboard(15),
			page:          10, // Invalid page, should default to max (2)
			expectedPage:  2,
			expectButtons: true,
			expectErr:     false,
		},
		{
			name: "Leaderboard with points display",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					embed := send.Embeds[0]
					if len(embed.Fields) != 1 {
						t.Errorf("Expected 1 field, got %d", len(embed.Fields))
					}

					// Entries with points should show "• N pts"
					expectedValue := "🥇 **Tag #1  ** <@user1> • 30 pts\n🥈 **Tag #2  ** <@user2> • 20 pts\n🥉 **Tag #3  ** <@user3> • 10 pts\n"
					if embed.Fields[0].Value != expectedValue {
						t.Errorf("Unexpected field value:\ngot:  %q\nwant: %q", embed.Fields[0].Value, expectedValue)
					}

					return &discordgo.Message{
						ID:      "test-message-id",
						Embeds:  send.Embeds,
						Content: "Test Message",
					}, nil
				}
			},
			leaderboard:   createTestLeaderboardWithPoints(3),
			page:          1,
			expectedPage:  1,
			expectButtons: false,
			expectErr:     false,
		},
		{
			name: "Discord API error",
			setupFake: func(t *testing.T, fakeSession *discord.FakeSession) {
				fakeSession.ChannelMessageSendComplexFunc = func(chID string, send *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, fmt.Errorf("discord API error")
				}
			},
			leaderboard: createTestLeaderboard(5),
			page:        1,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if tt.setupFake != nil {
				tt.setupFake(t, fakeSession)
			}

			lum := &leaderboardUpdateManager{
				logger:  mockLogger,
				session: fakeSession,
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

// Helper function to create test leaderboard entries with points
func createTestLeaderboardWithPoints(count int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = LeaderboardEntry{
			Rank:        sharedtypes.TagNumber(i + 1),
			UserID:      sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
			TotalPoints: (count - i) * 10,
		}
	}
	return entries
}
