package scoreround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func intPointer(i sharedtypes.Score) *sharedtypes.Score {
	return &i
}

func Test_scoreRoundManager_UpdateScoreEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	channelID := "testChannelID"
	messageID := "testMessageID"
	userID := sharedtypes.DiscordID("testUserID")

	tests := []struct {
		name               string
		setup              func(mockSession *discordmocks.MockSession, mockConfig *config.Config)
		score              *sharedtypes.Score
		expectError        bool
		expectedEmbedValue string
		expectedSuccessMsg string
	}{
		{
			name: "Successful Score Update",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ testNick", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					GuildMember(mockConfig.Discord.GuildID, string(userID)).
					Return(&discordgo.Member{User: &discordgo.User{Username: "testUser"}, Nick: "testNick"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
						if edit == nil {
							return nil, errors.New("edit struct is nil")
						}
						if edit.Embeds == nil {
							edit.Embeds = &[]*discordgo.MessageEmbed{}
						}
						updatedEmbeds := *edit.Embeds
						return &discordgo.Message{
							ID:        edit.ID,
							ChannelID: edit.Channel,
							Embeds:    updatedEmbeds,
						}, nil
					}).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "Score: +10",
			expectedSuccessMsg: "",
		},
		{
			name: "No Matching User in Embed",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ AnotherUser", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					GuildMember(mockConfig.Discord.GuildID, string(userID)).
					Return(&discordgo.Member{User: &discordgo.User{Username: "testUser"}, Nick: "testNick"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "User not found in embed",
		},
		{
			name: "Nil Score (Reset Score)",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ testNick", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					GuildMember(mockConfig.Discord.GuildID, string(userID)).
					Return(&discordgo.Member{User: &discordgo.User{Username: "testUser"}, Nick: "testNick"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
						updatedEmbeds := *edit.Embeds
						return &discordgo.Message{
							ID:        edit.ID,
							ChannelID: edit.Channel,
							Embeds:    updatedEmbeds,
						}, nil
					}).
					Times(1)
			},
			score:              nil,
			expectError:        false,
			expectedEmbedValue: "Score: --",
			expectedSuccessMsg: "",
		},
		{
			name: "Session Fails to Fetch Message",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(nil, errors.New("failed to fetch message")).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        true,
			expectedEmbedValue: "",
			expectedSuccessMsg: "",
		},
		{
			name: "Session Fails to Edit Message",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ testNick", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					GuildMember(mockConfig.Discord.GuildID, string(userID)).
					Return(&discordgo.Member{User: &discordgo.User{Username: "testUser"}, Nick: "testNick"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(nil, errors.New("failed to edit message")).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        true,
			expectedEmbedValue: "",
			expectedSuccessMsg: "",
		},
		{
			name: "User Fetch Fails",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ testNick", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(nil, errors.New("user fetch failed")).
					AnyTimes() // Use AnyTimes() to allow multiple calls
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "User not found in embed",
		},
		{
			name: "Guild Member Fetch Fails (Use Username)",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "🏌️ testUser", Value: "Score: 5"},
								},
							},
						},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					GuildMember(mockConfig.Discord.GuildID, string(userID)).
					Return(nil, errors.New("guild member fetch failed")).
					AnyTimes() // Use AnyTimes() to allow multiple calls

				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
						updatedEmbeds := *edit.Embeds
						return &discordgo.Message{
							ID:        edit.ID,
							ChannelID: edit.Channel,
							Embeds:    updatedEmbeds,
						}, nil
					}).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "Score: +10",
			expectedSuccessMsg: "",
		},
		{
			name: "Nil Embeds in Message",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID:     messageID,
						Embeds: nil,
					}, nil).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "User not found in embed",
		},
		{
			name: "Empty Embeds in Message",
			setup: func(mockSession *discordmocks.MockSession, mockConfig *config.Config) {
				mockSession.EXPECT().
					ChannelMessage(channelID, messageID).
					Return(&discordgo.Message{
						ID:     messageID,
						Embeds: []*discordgo.MessageEmbed{},
					}, nil).
					Times(1)
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "User not found in embed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := loggerfrolfbot.NewTestHandler()
			logger := slog.New(mockLogger)
			mockConfig := &config.Config{Discord: config.DiscordConfig{GuildID: "testGuildID"}}

			if tt.setup != nil {
				tt.setup(mockSession, mockConfig)
			}

			srm := &scoreRoundManager{
				session: mockSession,
				logger:  logger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			result, err := srm.UpdateScoreEmbed(ctx, channelID, messageID, userID, tt.score)

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("Expected result.Error to be non-nil but got nil")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Expected result.Error to be nil, but got: %v", result.Error)
				}
				if err != nil {
					t.Errorf("Expected returned error to be nil, but got: %v", err)
				}
			}

			if tt.expectedSuccessMsg != "" {
				if result.Success == nil || result.Success.(string) != tt.expectedSuccessMsg {
					t.Errorf("Expected success message %q, but got %v", tt.expectedSuccessMsg, result.Success)
				}
			} else if !tt.expectError {
				updatedMessage, ok := result.Success.(*discordgo.Message)
				if !ok {
					t.Errorf("Expected success result to be *discordgo.Message, but got %T", result.Success)
				} else {
					found := false
					user, userErr := mockSession.User(string(userID))
					if userErr == nil && user != nil {
						username := user.Username
						if member, memberErr := mockSession.GuildMember(mockConfig.Discord.GuildID, string(userID)); memberErr == nil && member.Nick != "" {
							username = member.Nick
						}
						targetFieldName := fmt.Sprintf("🏌️ %s", username)

						for _, embed := range updatedMessage.Embeds {
							if embed == nil {
								continue
							}
							for _, field := range embed.Fields {
								if field.Name == targetFieldName {
									if field.Value != tt.expectedEmbedValue {
										t.Errorf("Expected embed value %q, but got %q", tt.expectedEmbedValue, field.Value)
									}
									found = true
									break
								}
							}
							if found {
								break
							}
						}

						if !found && tt.expectedEmbedValue != "" {
							t.Errorf("Expected embed field with value %q to be updated but it was not found", tt.expectedEmbedValue)
						}
					}
				}
			}

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("Expected an error in result, but got nil")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Expected no error in result, but got: %v", result.Error)
				}
			}
		})
	}
}

func TestUpdateUserScoreInEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := sharedtypes.DiscordID("testUserID")
	guildID := "testGuildID"

	tests := []struct {
		name          string
		setup         func(mockSession *discordmocks.MockSession)
		embed         *discordgo.MessageEmbed
		score         *sharedtypes.Score
		userNick      string
		userName      string
		expectResult  bool
		expectedValue string
	}{
		{
			name: "Successfully Update Score With Nickname",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes()
				mockSession.EXPECT().
					GuildMember(guildID, string(userID)).
					Return(&discordgo.Member{Nick: "testNick"}, nil).
					AnyTimes()
			},
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testNick", Value: "Score: 5"},
				},
			},
			score:         intPointer(10),
			userNick:      "testNick",
			expectResult:  true,
			expectedValue: "Score: +10",
		},
		{
			name: "Successfully Update Score With Username",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes()
				mockSession.EXPECT().
					GuildMember(guildID, string(userID)).
					Return(nil, errors.New("no guild member")).
					AnyTimes()
			},
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testUser", Value: "Score: 5"},
				},
			},
			score:         intPointer(10),
			userName:      "testUser",
			expectResult:  true,
			expectedValue: "Score: +10",
		},
		{
			name: "Reset Score To Default",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes()
				mockSession.EXPECT().
					GuildMember(guildID, string(userID)).
					Return(&discordgo.Member{Nick: "testNick"}, nil).
					AnyTimes()
			},
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testNick", Value: "Score: 5"},
				},
			},
			score:         nil,
			userNick:      "testNick",
			expectResult:  true,
			expectedValue: "Score: --",
		},
		{
			name: "User Not Found In Embed",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(string(userID)).
					Return(&discordgo.User{ID: string(userID), Username: "testUser"}, nil).
					AnyTimes()
				mockSession.EXPECT().
					GuildMember(guildID, string(userID)).
					Return(&discordgo.Member{Nick: "testNick"}, nil).
					AnyTimes()
			},
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ anotherUser", Value: "Score: 5"},
				},
			},
			score:        intPointer(10),
			expectResult: false,
		},
		{
			name:         "Nil Embed",
			setup:        func(mockSession *discordmocks.MockSession) {},
			embed:        nil,
			score:        intPointer(10),
			expectResult: false,
		},
		{
			name: "Error Fetching User",
			setup: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(string(userID)).
					Return(nil, errors.New("failed to fetch user")).
					Times(1)
			},
			embed: &discordgo.MessageEmbed{
				Title: "Scorecard",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "🏌️ testUser", Value: "Score: 5"},
				},
			},
			score:        intPointer(10),
			expectResult: false,
		},
		{
			name:  "Empty Fields In Embed",
			setup: func(mockSession *discordmocks.MockSession) {},
			embed: &discordgo.MessageEmbed{
				Title:  "Scorecard",
				Fields: []*discordgo.MessageEmbedField{},
			},
			score:        intPointer(10),
			expectResult: false,
		},
		{
			name:  "Nil Fields In Embed",
			setup: func(mockSession *discordmocks.MockSession) {},
			embed: &discordgo.MessageEmbed{
				Title:  "Scorecard",
				Fields: nil,
			},
			score:        intPointer(10),
			expectResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSession := discordmocks.NewMockSession(ctrl)

			if tt.setup != nil {
				tt.setup(mockSession)
			}

			result := UpdateUserScoreInEmbed(ctx, mockSession, tt.embed, userID, tt.score, guildID)

			if result != tt.expectResult {
				t.Errorf("UpdateUserScoreInEmbed() = %v, want %v", result, tt.expectResult)
			}

			if tt.expectResult && tt.embed != nil && tt.embed.Fields != nil {
				username := tt.userNick
				if username == "" {
					username = tt.userName
				}
				targetFieldName := fmt.Sprintf("🏌️ %s", username)

				found := false
				for _, field := range tt.embed.Fields {
					if field.Name == targetFieldName {
						if field.Value != tt.expectedValue {
							t.Errorf("Expected embed value %q, but got %q", tt.expectedValue, field.Value)
						}
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected to find and update embed field but no matching field was found")
				}
			}
		})
	}
}
