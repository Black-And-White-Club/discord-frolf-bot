package startround

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_startRoundManager_TransformRoundToScorecard(t *testing.T) {
	// Helper function to create a pointer to a string
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a pointer to a time
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Create fixed time for testing
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	testRoundID := sharedtypes.RoundID(uuid.New())

	// Test cases
	tests := []struct {
		name               string
		setup              func(mockSession *discordmocks.MockSession)
		payload            *roundevents.DiscordRoundStartPayload
		expectedEmbed      *discordgo.MessageEmbed
		expectedComponents []discordgo.MessageComponent
		expectError        bool
	}{
		{
			name: "Basic round with no participants",
			setup: func(mockSession *discordmocks.MockSession) {
				// No setup needed for this case
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "📅 Started",
						Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix()),
					},
					{
						Name:  "📍 Location",
						Value: "Test Course",
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Enter Score",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("round_enter_score|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("round_join_late|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with accepted participants",
			setup: func(mockSession *discordmocks.MockSession) {
				// No mock calls needed since the function uses Discord mentions directly
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{
					{
						UserID:    "user-123",
						TagNumber: nil,
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
					{
						UserID:    "user-456",
						TagNumber: nil,
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "📅 Started",
						Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix()),
					},
					{
						Name:  "📍 Location",
						Value: "Test Course",
					},
					{
						Name:   "✅ Accepted",
						Value:  "<@user-123> — Score: --\n<@user-456> — Score: --",
						Inline: false,
					},
					{
						Name:   "🤔 Tentative",
						Value:  "*No participants*",
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Enter Score",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("round_enter_score|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("round_join_late|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with mixed participant responses",
			setup: func(mockSession *discordmocks.MockSession) {
				// No mock calls needed
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   testRoundID,
				Title:     "Mixed Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{
					{
						UserID:    "user-123",
						TagNumber: nil,
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
					{
						UserID:    "user-456",
						TagNumber: nil,
						Score:     nil,
						Response:  roundtypes.ResponseTentative,
					},
					{
						UserID:    "user-789",
						TagNumber: nil,
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Mixed Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "📅 Started",
						Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix()),
					},
					{
						Name:  "📍 Location",
						Value: "Test Course",
					},
					{
						Name:   "✅ Accepted",
						Value:  "<@user-123> — Score: --\n<@user-789> — Score: --",
						Inline: false,
					},
					{
						Name:   "🤔 Tentative",
						Value:  "<@user-456> — Score: --",
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Enter Score",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("round_enter_score|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("round_join_late|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with participants with tag numbers",
			setup: func(mockSession *discordmocks.MockSession) {
				// No mock calls needed
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   testRoundID,
				Title:     "Tagged Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{
					{
						UserID:    "user-123",
						TagNumber: func() *sharedtypes.TagNumber { t := sharedtypes.TagNumber(1); return &t }(),
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
					{
						UserID:    "user-456",
						TagNumber: func() *sharedtypes.TagNumber { t := sharedtypes.TagNumber(2); return &t }(),
						Score:     nil,
						Response:  roundtypes.ResponseAccept,
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Tagged Round** - Round Started",
				Description: "Round at Test Course has started!",
				Color:       0x00AA00,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "📅 Started",
						Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix()),
					},
					{
						Name:  "📍 Location",
						Value: "Test Course",
					},
					{
						Name:   "✅ Accepted",
						Value:  "<@user-123> Tag: 1 — Score: --\n<@user-456> Tag: 2 — Score: --",
						Inline: false,
					},
					{
						Name:   "🤔 Tentative",
						Value:  "*No participants*",
						Inline: false,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Enter Score",
							Style:    discordgo.PrimaryButton,
							CustomID: fmt.Sprintf("round_enter_score|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("round_join_late|%s", testRoundID),
							Emoji: &discordgo.ComponentEmoji{
								Name: "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "guild-id",
				},
			}

			// Call the setup function for the test case
			if tt.setup != nil {
				tt.setup(mockSession)
			}

			// Create manager with mocks and bypass the operationWrapper
			srm := &startRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			// Call the function
			result, err := srm.TransformRoundToScorecard(context.Background(), tt.payload, nil)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error, got nil")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// If we don't expect an error, validate the results
			if !tt.expectError {
				// Check if the result is successful
				if result.Error != nil {
					t.Errorf("Unexpected error in result: %v", result.Error)
					return
				}

				// Extract the success data
				successData, ok := result.Success.(struct {
					Embed      *discordgo.MessageEmbed
					Components []discordgo.MessageComponent
				})
				if !ok {
					t.Errorf("Failed to cast result.Success to expected type")
					return
				}

				gotEmbed := successData.Embed
				gotComponents := successData.Components

				// For timestamp, we just check that it's set to something valid
				if gotEmbed.Timestamp == "" {
					t.Errorf("Expected timestamp to be set, got empty string")
				}

				// Clear timestamp for comparison since it's dynamic
				origTimestamp := gotEmbed.Timestamp
				gotEmbed.Timestamp = ""

				// Do detailed field comparison
				if gotEmbed.Title != tt.expectedEmbed.Title {
					t.Errorf("Title mismatch: got %q, want %q", gotEmbed.Title, tt.expectedEmbed.Title)
				}
				if gotEmbed.Description != tt.expectedEmbed.Description {
					t.Errorf("Description mismatch: got %q, want %q", gotEmbed.Description, tt.expectedEmbed.Description)
				}
				if gotEmbed.Color != tt.expectedEmbed.Color {
					t.Errorf("Color mismatch: got %d, want %d", gotEmbed.Color, tt.expectedEmbed.Color)
				}

				// Compare fields
				if len(gotEmbed.Fields) != len(tt.expectedEmbed.Fields) {
					t.Errorf("Fields length mismatch: got %d, want %d", len(gotEmbed.Fields), len(tt.expectedEmbed.Fields))
				} else {
					for i, gotField := range gotEmbed.Fields {
						expectedField := tt.expectedEmbed.Fields[i]
						if gotField.Name != expectedField.Name {
							t.Errorf("Field[%d] Name mismatch: got %q, want %q", i, gotField.Name, expectedField.Name)
						}
						if gotField.Value != expectedField.Value {
							t.Errorf("Field[%d] Value mismatch: got %q, want %q", i, gotField.Value, expectedField.Value)
						}
						if gotField.Inline != expectedField.Inline {
							t.Errorf("Field[%d] Inline mismatch: got %t, want %t", i, gotField.Inline, expectedField.Inline)
						}
					}
				}

				// Compare footer
				if !reflect.DeepEqual(gotEmbed.Footer, tt.expectedEmbed.Footer) {
					t.Errorf("Footer mismatch: got %+v, want %+v", gotEmbed.Footer, tt.expectedEmbed.Footer)
				}

				// Compare components
				if !reflect.DeepEqual(gotComponents, tt.expectedComponents) {
					t.Errorf("Components mismatch: got %+v, want %+v", gotComponents, tt.expectedComponents)
				}

				// Set timestamp back after tests
				gotEmbed.Timestamp = origTimestamp
			}
		})
	}
}
