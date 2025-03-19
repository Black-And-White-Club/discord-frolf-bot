package scoreround

import (
	"context"
	"errors"
	"fmt"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_scoreRoundManager_UpdateScoreEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	channelID := "testChannelID"
	messageID := "testMessageID"
	userID := roundtypes.UserID("testUserID")

	tests := []struct {
		name               string
		initialEmbeds      []*discordgo.MessageEmbed
		score              *int
		mockMessageError   error
		mockEditError      error
		expectEditCall     bool
		expectUpdate       bool
		expectError        bool
		expectedEmbedValue string
	}{
		{
			name: "Successful Score Update",
			initialEmbeds: []*discordgo.MessageEmbed{
				{
					Title: "Scorecard",
					Fields: []*discordgo.MessageEmbedField{
						{Name: "🏌️ testNick", Value: "Score: 5"},
					},
				},
			},
			score:              intPointer(10),
			expectEditCall:     true,
			expectUpdate:       true,
			expectedEmbedValue: "Score: +10",
		},
		{
			name: "No Matching User in Embed",
			initialEmbeds: []*discordgo.MessageEmbed{
				{
					Title: "Scorecard",
					Fields: []*discordgo.MessageEmbedField{
						{Name: "🏌️ AnotherUser", Value: "Score: 5"},
					},
				},
			},
			score:          intPointer(10),
			expectEditCall: false,
			expectUpdate:   false,
		},
		{
			name: "Nil Score (Reset Score)",
			initialEmbeds: []*discordgo.MessageEmbed{
				{
					Title: "Scorecard",
					Fields: []*discordgo.MessageEmbedField{
						{Name: "🏌️ testNick", Value: "Score: 5"},
					},
				},
			},
			score:              nil,
			expectEditCall:     true,
			expectUpdate:       true,
			expectedEmbedValue: "Score: --",
		},
		{
			name:             "Session Fails to Fetch Message",
			mockMessageError: errors.New("failed to fetch message"),
			score:            intPointer(10),
			expectEditCall:   false,
			expectError:      true,
		},
		{
			name: "Session Fails to Edit Message",
			initialEmbeds: []*discordgo.MessageEmbed{
				{
					Title: "Scorecard",
					Fields: []*discordgo.MessageEmbedField{
						{Name: "🏌️ testNick", Value: "Score: 5"},
					},
				},
			},
			score:          intPointer(10),
			mockEditError:  errors.New("failed to edit message"),
			expectEditCall: true,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := observability.NewNoOpLogger()
			mockConfig := &config.Config{Discord: config.DiscordConfig{GuildID: "testGuildID"}}

			srm := &scoreRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			// Mock fetching the original message
			mockSession.EXPECT().
				ChannelMessage(channelID, messageID).
				Return(&discordgo.Message{
					ID:     messageID,
					Embeds: tt.initialEmbeds,
				}, tt.mockMessageError).
				AnyTimes()

			// Mock fetching the user
			mockSession.EXPECT().
				User(string(userID)).
				Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
				AnyTimes()

			// Mock fetching the guild member
			mockSession.EXPECT().
				GuildMember(mockConfig.Discord.GuildID, string(userID)).
				Return(&discordgo.Member{User: &discordgo.User{Username: "testUser"}, Nick: "testNick"}, nil).
				AnyTimes()

			if tt.expectEditCall {
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
						if edit == nil {
							return nil, errors.New("edit struct is nil")
						}
						if edit.Embeds == nil {
							edit.Embeds = &[]*discordgo.MessageEmbed{}
						}
						return &discordgo.Message{
							ID:        edit.ID,
							ChannelID: edit.Channel,
							Embeds:    *edit.Embeds,
						}, tt.mockEditError
					}).
					Times(1)
			}

			// Run the function
			updatedMessage, err := srm.UpdateScoreEmbed(ctx, channelID, messageID, userID, tt.score)

			// Assertions
			if tt.expectError && err == nil {
				t.Errorf("Expected error, but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if tt.expectUpdate {
				found := false
				for _, embed := range updatedMessage.Embeds {
					for _, field := range embed.Fields {
						if field.Name == "🏌️ testNick" {
							if field.Value != tt.expectedEmbedValue {
								t.Errorf("Expected embed value %q, but got %q", tt.expectedEmbedValue, field.Value)
							}
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Expected embed field to be updated but it was not found")
				}
			}
		})
	}
}

// Helper function to return a pointer to an int
func intPointer(i int) *int {
	return &i
}

func TestUpdateUserScoreInEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := "testUserID"
	guildID := "testGuildID"

	tests := []struct {
		name            string
		embed           *discordgo.MessageEmbed
		score           *int
		mockUserError   error
		mockMemberError error
		userNick        string
		userName        string
		expectResult    bool
		expectedValue   string
	}{
		{
			name: "Successfully Update Score With Nick",
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testNick", Value: "Score: 5"},
				},
			},
			score:         intPointer(10),
			userNick:      "testNick",
			userName:      "testUser",
			expectResult:  true,
			expectedValue: "Score: +10",
		},
		{
			name: "Successfully Update Score With Username",
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testUser", Value: "Score: 5"},
				},
			},
			score:           intPointer(10),
			mockMemberError: errors.New("no guild member"),
			userName:        "testUser",
			expectResult:    true,
			expectedValue:   "Score: +10",
		},
		{
			name: "Reset Score To Default",
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testNick", Value: "Score: 5"},
				},
			},
			score:         nil,
			userNick:      "testNick",
			userName:      "testUser",
			expectResult:  true,
			expectedValue: "Score: --",
		},
		{
			name: "User Not Found In Embed",
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ anotherUser", Value: "Score: 5"},
				},
			},
			score:        intPointer(10),
			userNick:     "testNick",
			userName:     "testUser",
			expectResult: false,
		},
		{
			name:         "Nil Embed",
			embed:        nil,
			score:        intPointer(10),
			expectResult: false,
		},
		{
			name: "Error Fetching User",
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testUser", Value: "Score: 5"},
				},
			},
			score:         intPointer(10),
			mockUserError: errors.New("failed to fetch user"),
			expectResult:  false,
		},
		{
			name: "Empty Fields In Embed",
			embed: &discordgo.MessageEmbed{
				Title:  "Scorecard",
				Fields: []*discordgo.MessageEmbedField{},
			},
			score:        intPointer(10),
			expectResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSession := discordmocks.NewMockSession(ctrl)

			// Mock fetching the user
			if tt.mockUserError != nil {
				mockSession.EXPECT().
					User(userID).
					Return(nil, tt.mockUserError).
					Times(1)
			} else if tt.embed != nil && len(tt.embed.Fields) > 0 {
				mockSession.EXPECT().
					User(userID).
					Return(&discordgo.User{ID: userID, Username: tt.userName}, nil).
					AnyTimes()

				// Mock fetching the guild member only if there's no user error
				if tt.mockMemberError != nil {
					mockSession.EXPECT().
						GuildMember(guildID, userID).
						Return(nil, tt.mockMemberError).
						AnyTimes()
				} else {
					mockSession.EXPECT().
						GuildMember(guildID, userID).
						Return(&discordgo.Member{
							User: &discordgo.User{Username: tt.userName},
							Nick: tt.userNick,
						}, nil).
						AnyTimes()
				}
			}

			// Run the function
			result := UpdateUserScoreInEmbed(ctx, mockSession, tt.embed, userID, tt.score, guildID)

			// Check if result matches expected
			if result != tt.expectResult {
				t.Errorf("UpdateUserScoreInEmbed() = %v, want %v", result, tt.expectResult)
			}

			// If update was expected, verify the field value
			if tt.expectResult && tt.embed != nil {
				var targetName string
				if tt.mockMemberError != nil {
					targetName = fmt.Sprintf("🏌️ %s", tt.userName)
				} else {
					targetName = fmt.Sprintf("🏌️ %s", tt.userNick)
				}

				found := false
				for _, field := range tt.embed.Fields {
					if field.Name == targetName {
						if field.Value != tt.expectedValue {
							t.Errorf("Expected embed value %q, but got %q", tt.expectedValue, field.Value)
						}
						found = true
						break
					}
				}

				if !found && tt.expectResult {
					t.Errorf("Expected to find and update embed field but no matching field was found")
				}
			}
		})
	}
}
