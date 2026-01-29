package scoreround

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func intPointer(i sharedtypes.Score) *sharedtypes.Score {
	return &i
}

func Test_scoreRoundManager_UpdateScoreEmbed(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	ctx := context.Background()
	channelID := "testChannelID"
	messageID := "testMessageID"
	userID := sharedtypes.DiscordID("123456789012345678") // Use valid numeric Discord ID

	tests := []struct {
		name               string
		setup              func()
		score              *sharedtypes.Score
		expectError        bool
		expectedEmbedValue string
		expectedSuccessMsg string
	}{
		{
			name: "Successful Score Update",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "✅ Accepted", Value: "<@123456789012345678> Tag: 1 — Score: 5"},
								},
							},
						},
					}, nil
				}

				fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
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
				}
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "<@123456789012345678> Tag: 1 — Score: +10",
			expectedSuccessMsg: "",
		},
		{
			name: "Nil Score (Reset Score)",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "✅ Accepted", Value: "<@123456789012345678> Tag: 1 — Score: 5"},
								},
							},
						},
					}, nil
				}

				fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					updatedEmbeds := *edit.Embeds
					return &discordgo.Message{
						ID:        edit.ID,
						ChannelID: edit.Channel,
						Embeds:    updatedEmbeds,
					}, nil
				}
			},
			score:              nil,
			expectError:        false,
			expectedEmbedValue: "<@123456789012345678> Tag: 1 — Score: --",
			expectedSuccessMsg: "",
		},
		{
			name: "Session Fails to Edit Message",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "✅ Accepted", Value: "<@123456789012345678> Tag: 1 — Score: 5"},
								},
							},
						},
					}, nil
				}

				fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("failed to edit message")
				}
			},
			score:              intPointer(10),
			expectError:        true,
			expectedEmbedValue: "",
			expectedSuccessMsg: "",
		},
		{
			name: "Guild Member Fetch Fails (Use Username)",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "✅ Accepted", Value: "<@123456789012345678> Tag: 1 — Score: 5"},
								},
							},
						},
					}, nil
				}

				fakeSession.ChannelMessageEditComplexFunc = func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					updatedEmbeds := *edit.Embeds
					return &discordgo.Message{
						ID:        edit.ID,
						ChannelID: edit.Channel,
						Embeds:    updatedEmbeds,
					}, nil
				}
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "<@123456789012345678> Tag: 1 — Score: +10",
			expectedSuccessMsg: "",
		},
		{
			name: "User Fetch Fails",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID: messageID,
						Embeds: []*discordgo.MessageEmbed{
							{
								Title: "Scorecard",
								Fields: []*discordgo.MessageEmbedField{
									{Name: "✅ Accepted", Value: "<@987654321098765432> Tag: 1 — Score: 5"}, // Different user ID
								},
							},
						},
					}, nil
				}
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "User 123456789012345678 not found in embed fields to update score",
		},
		{
			name: "Nil Embeds in Message",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     messageID,
						Embeds: nil,
					}, nil
				}
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "No embeds found to update",
		},
		{
			name: "Empty Embeds in Message",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(cID, mID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     messageID,
						Embeds: []*discordgo.MessageEmbed{},
					}, nil
				}
			},
			score:              intPointer(10),
			expectError:        false,
			expectedEmbedValue: "",
			expectedSuccessMsg: "No embeds found to update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			srm := &scoreRoundManager{
				session: fakeSession,
				logger:  loggerfrolfbot.NoOpLogger,
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			result, _ := srm.UpdateScoreEmbed(ctx, channelID, messageID, userID, tt.score)

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("Expected result.Error to be non-nil but got nil")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Expected no error in result, but got: %v", result.Error)
				}
			}

			if tt.expectedSuccessMsg != "" {
				if result.Success == nil {
					t.Errorf("Expected success message %q, but got nil", tt.expectedSuccessMsg)
				} else if successStr, ok := result.Success.(string); !ok || successStr != tt.expectedSuccessMsg {
					t.Errorf("Expected success message %q, but got %v", tt.expectedSuccessMsg, result.Success)
				}
			} else if !tt.expectError {
				updatedMessage, ok := result.Success.(*discordgo.Message)
				if !ok {
					t.Errorf("Expected success result to be *discordgo.Message, but got %T: %v", result.Success, result.Success)
				} else if tt.expectedEmbedValue != "" {
					// Verify the embed was updated correctly
					found := false
					for _, embed := range updatedMessage.Embeds {
						if embed == nil {
							continue
						}
						for _, field := range embed.Fields {
							if strings.Contains(field.Value, string(userID)) {
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

					if !found {
						t.Errorf("Expected embed field with value %q to be updated but it was not found", tt.expectedEmbedValue)
					}
				}
			}
		})
	}
}
