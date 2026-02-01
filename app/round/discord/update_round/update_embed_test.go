package updateround

import (
	"context"
	"fmt"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// helper to create a mock embed
func createMockEmbed(title, description, location string, timestamp time.Time, participants []string) *discordgo.MessageEmbed {
	unix := timestamp.Unix()
	fields := []*discordgo.MessageEmbedField{
		{Name: "📅 Time", Value: fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", unix, unix)},
		{Name: "📍 Location", Value: location},
	}
	for _, p := range participants {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Participant", Value: p})
	}
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Fields:      fields,
		Timestamp:   timestamp.Format(time.RFC3339),
	}
}

func Test_updateRoundManager_UpdateRoundEventEmbed(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	timePtr := func(t time.Time) *time.Time { return &t }
	locPtr := func(s string) *roundtypes.Location { l := roundtypes.Location(s); return &l }

	channelID := "test-channel"
	testRoundID := sharedtypes.RoundID(uuid.New())
	testRoundIDStr := uuid.UUID(testRoundID).String()
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	participants := []string{"<@user1>"}
	originalEmbed := createMockEmbed("Original Title", "Original Description", "Original Location", fixedTime, participants)

	tests := []struct {
		name         string
		title        *roundtypes.Title
		description  *roundtypes.Description
		startTime    *sharedtypes.StartTime
		location     *roundtypes.Location
		setupSession func(fs *discord.FakeSession)
		expectErr    bool
		check        func(embed *discordgo.MessageEmbed)
	}{
		{
			name:  "Update title",
			title: (*roundtypes.Title)(strPtr("New Title")),
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
				fs.ChannelMessageEditEmbedFunc = func(chID, msgID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
				}
			},
			check: func(embed *discordgo.MessageEmbed) {
				if embed.Title != "**New Title**" {
					t.Errorf("Expected title '**New Title**', got %s", embed.Title)
				}
			},
		},
		{
			name:        "Update description",
			description: (*roundtypes.Description)(strPtr("Updated Description")),
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
				fs.ChannelMessageEditEmbedFunc = func(chID, msgID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
				}
			},
			check: func(embed *discordgo.MessageEmbed) {
				if embed.Description != "Updated Description" {
					t.Errorf("Expected description 'Updated Description', got %s", embed.Description)
				}
			},
		},
		{
			name:      "Update start time",
			startTime: (*sharedtypes.StartTime)(timePtr(time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC))),
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
				fs.ChannelMessageEditEmbedFunc = func(chID, msgID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
				}
			},
			check: func(embed *discordgo.MessageEmbed) {
				expectedUnix := int64(1672531201)
				expectedValue := fmt.Sprintf("<t:%d:f>  (**Starts**: <t:%d:R>)", expectedUnix, expectedUnix)
				if embed.Fields[0].Value != expectedValue {
					t.Errorf("Unexpected time field: got %s, want %s", embed.Fields[0].Value, expectedValue)
				}
			},
		},
		{
			name:     "Update location",
			location: locPtr("Updated Location"),
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
				fs.ChannelMessageEditEmbedFunc = func(chID, msgID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
				}
			},
			check: func(embed *discordgo.MessageEmbed) {
				if embed.Fields[1].Value != "Updated Location" {
					t.Errorf("Expected location 'Updated Location', got %s", embed.Fields[1].Value)
				}
			},
		},
		{
			name: "No updates provided",
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
			},
			check:     func(embed *discordgo.MessageEmbed) {},
			expectErr: false,
		},
		{
			name:        "Multiple updates",
			title:       (*roundtypes.Title)(strPtr("New Title")),
			description: (*roundtypes.Description)(strPtr("Updated Description")),
			startTime:   (*sharedtypes.StartTime)(timePtr(time.Date(2025, 5, 5, 12, 0, 0, 0, time.UTC))),
			location:    locPtr("New Location"),
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{originalEmbed}}, nil
				}
				fs.ChannelMessageEditEmbedFunc = func(chID, msgID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{embed}}, nil
				}
			},
			check: func(embed *discordgo.MessageEmbed) {
				if embed.Title != "**New Title**" || embed.Description != "Updated Description" || embed.Fields[1].Value != "New Location" {
					t.Errorf("Embed fields not updated correctly")
				}
			},
		},
		{
			name: "No embeds in message",
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{Embeds: []*discordgo.MessageEmbed{}}, nil
				}
			},
			expectErr: false,
			check:     nil,
		},
		{
			name: "Channel message fetch fails",
			setupSession: func(fs *discord.FakeSession) {
				fs.ChannelMessageFunc = func(chID, msgID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, fmt.Errorf("fetch error")
				}
			},
			expectErr: true,
			check:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()
			tt.setupSession(fakeSession)

			urm := &updateRoundManager{
				session: fakeSession,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			ctx := context.Background()
			got, err := urm.UpdateRoundEventEmbed(ctx, channelID, testRoundIDStr, tt.title, tt.description, tt.startTime, tt.location)

			if (err != nil) != tt.expectErr {
				t.Errorf("UpdateRoundEventEmbed() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			// Special-case: when there are no embeds, the implementation returns nil error
			// but places the error inside the result.Error field. Assert that here.
			if tt.name == "No embeds in message" {
				if got.Error == nil {
					t.Errorf("expected result.Error to be non-nil when no embeds are present")
				}
				return
			}

			if tt.check != nil && got.Success != nil {
				msg, ok := got.Success.(*discordgo.Message)
				if !ok {
					t.Fatalf("Success type not *discordgo.Message")
				}
				tt.check(msg.Embeds[0])
			}
		})
	}
}
