package updateround

import (
	"context"
	"fmt"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func createMockEmbed(title, description, location string, timestamp time.Time, participants []string) *discordgo.MessageEmbed {
	unixTimestamp := timestamp.Unix() // Get the Unix timestamp

	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "📅 Time",
			Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unixTimestamp, unixTimestamp), // Use dynamic timestamp
		},
		{
			Name:  "📍 Location",
			Value: location,
		},
	}

	for _, participant := range participants {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Participant",
			Value: participant,
		})
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Timestamp:   timestamp.Format(time.RFC3339),
		Fields:      fields,
	}
}

func Test_updateRoundManager_UpdateRoundEventEmbed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testRoundIDString := uuid.UUID(testRoundID).String()
	channelID := "test-channel"

	strPtr := func(s string) *string {
		return &s
	}

	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		setupMocks    func(mockSession *discordmocks.MockSession)
		title         *roundtypes.Title
		description   *roundtypes.Description
		startTime     *sharedtypes.StartTime
		location      *roundtypes.Location
		originalEmbed *discordgo.MessageEmbed
		expectedEmbed *discordgo.MessageEmbed
		expectErr     bool
	}{
		{
			name: "Update title",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()). // Updated to include the fourth parameter
					DoAndReturn(func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the title is updated correctly
						if embed.Title != "**New Title**" {
							t.Errorf("Unexpected title: got %s, want %s", embed.Title, "**New Title**")
						}
						// Return the updated message with the embed
						return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
					}).
					Times(1)
			},
			title:         (*roundtypes.Title)(strPtr("New Title")),
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("**New Title**", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectErr:     false,
		},
		{
			name: "Update description",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()). // Updated to include the fourth parameter
					DoAndReturn(func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the description is updated correctly
						if embed.Description != "Updated Description" {
							t.Errorf("Unexpected description: got %s, want %s", embed.Description, "Updated Description")
						}
						// Return the updated message with the embed
						return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
					}).
					Times(1)
			},
			description:   (*roundtypes.Description)(strPtr("Updated Description")),
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("Original Title", "Updated Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectErr:     false,
		},
		{
			name: "Update start time",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()). // Updated to include the fourth parameter
					DoAndReturn(func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the time field is updated correctly
						if embed.Fields[0].Value != "<t:1672531201:f>  (**Starts**: <t:1672531201:R>)" {
							t.Errorf("Unexpected time field: got %s, want %s", embed.Fields[0].Value, "<t:1672531201:f>  (**Starts**: <t:1672531201:R>)")
						}
						// Return the updated message with the embed
						return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
					}).
					Times(1)
			},
			startTime:     (*sharedtypes.StartTime)(timePtr(time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC))), // Set to the expected time
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC), []string{"<@user1>"}),
			expectErr:     false,
		},
		{
			name: "Update location",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()). // Updated to include the fourth parameter
					DoAndReturn(func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Check if the location field is updated correctly
						if embed.Fields[1].Value != "Updated Location" {
							t.Errorf("Unexpected location field: got %s, want %s", embed.Fields[1].Value, "Updated Location")
						}
						// Return the updated message with the embed
						return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
					}).
					Times(1)
			},
			location:      (*roundtypes.Location)(strPtr("Updated Location")),
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("Original Title", "Original Description", "Updated Location", fixedTime, []string{"<@user1>"}),
			expectErr:     false,
		},
		{
			name: "No updates provided",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()). // Updated to include the fourth parameter
					Times(0)                                                                         // No edit should occur
			},
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectErr:     false,
		},
		{
			name: "Participant unchanged",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					ChannelMessage(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"})}}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						if len(embed.Fields) != 3 {
							t.Errorf("Unexpected number of fields: got %d, want 3", len(embed.Fields))
						}
						// Return the updated message with the embed
						return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
					}).
					Times(1)
			},
			title:         (*roundtypes.Title)(strPtr("New Title")),
			originalEmbed: createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectedEmbed: createMockEmbed("**New Title**", "Original Description", "Original Location", fixedTime, []string{"<@user1>"}),
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			urm := &updateRoundManager{
				session: mockSession,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			ctx := context.Background()
			got, err := urm.UpdateRoundEventEmbed(ctx, channelID, testRoundIDString, tt.title, tt.description, tt.startTime, tt.location)

			if (err != nil) != tt.expectErr {
				t.Errorf("UpdateRoundEventEmbed() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				// Verify the updated embed matches the expected embed
				updatedMsg, ok := got.Success.(*discordgo.Message)
				if !ok {
					t.Errorf("UpdateRoundEventEmbed() Success is not a *discordgo.Message")
					return
				}

				if len(updatedMsg.Embeds) != 1 {
					t.Errorf("UpdateRoundEventEmbed() updated message has %d embeds, want 1", len(updatedMsg.Embeds))
					return
				}

				updatedEmbed := updatedMsg.Embeds[0]
				if updatedEmbed.Title != tt.expectedEmbed.Title ||
					updatedEmbed.Description != tt.expectedEmbed.Description ||
					updatedEmbed.Timestamp != tt.expectedEmbed.Timestamp {
					t.Errorf("UpdateRoundEventEmbed() updated embed mismatch: got %+v, want %+v", updatedEmbed, tt.expectedEmbed)
				}

				if len(updatedEmbed.Fields) != len(tt.expectedEmbed.Fields) {
					t.Errorf("UpdateRoundEventEmbed() updated embed fields count mismatch: got %d, want %d", len(updatedEmbed.Fields), len(tt.expectedEmbed.Fields))
				} else {
					for i := range updatedEmbed.Fields {
						if updatedEmbed.Fields[i].Name != tt.expectedEmbed.Fields[i].Name ||
							updatedEmbed.Fields[i].Value != tt.expectedEmbed.Fields[i].Value {
							t.Errorf("UpdateRoundEventEmbed() updated embed field %d mismatch: got %+v, want %+v", i, updatedEmbed.Fields[i], tt.expectedEmbed.Fields[i])
						}
					}
				}
			}
		})
	}
}
