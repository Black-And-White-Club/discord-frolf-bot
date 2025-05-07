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
				// Timestamp will be checked separately
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
			name: "Round with participants (no nicknames)",
			setup: func(mockSession *discordmocks.MockSession) {
				// Mock User calls
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{Username: "TestUser1"}, nil).
					Times(1)

				mockSession.EXPECT().
					User("user-456").
					Return(&discordgo.User{Username: "TestUser2"}, nil).
					Times(1)

				// Mock GuildMember calls - no nicknames
				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: ""}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-456").
					Return(&discordgo.Member{Nick: ""}, nil).
					Times(1)
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
					},
					{
						UserID:    "user-456",
						TagNumber: nil,
						Score:     nil,
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
						Name:   "🏌️ TestUser1",
						Value:  "Score: --",
						Inline: true,
					},
					{
						Name:   "🏌️ TestUser2",
						Value:  "Score: --",
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round in progress. Use the buttons below to join or record your score.",
				},
				// Timestamp will be checked separately
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
		// Add other test cases here...
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
			result, err := srm.TransformRoundToScorecard(context.Background(), tt.payload, tt.expectedEmbed)

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

				// Do field by field comparison except timestamp
				if !reflect.DeepEqual(gotEmbed.Title, tt.expectedEmbed.Title) {
					t.Errorf("Title mismatch: got %v, want %v", gotEmbed.Title, tt.expectedEmbed.Title)
				}
				if !reflect.DeepEqual(gotEmbed.Description, tt.expectedEmbed.Description) {
					t.Errorf("Description mismatch: got %v, want %v", gotEmbed.Description, tt.expectedEmbed.Description)
				}
				if !reflect.DeepEqual(gotEmbed.Color, tt.expectedEmbed.Color) {
					t.Errorf("Color mismatch: got %v, want %v", gotEmbed.Color, tt.expectedEmbed.Color)
				}
				if !reflect.DeepEqual(gotEmbed.Fields, tt.expectedEmbed.Fields) {
					t.Errorf("Fields mismatch: got %v, want %v", gotEmbed.Fields, tt.expectedEmbed.Fields)
				}
				if !reflect.DeepEqual(gotEmbed.Footer, tt.expectedEmbed.Footer) {
					t.Errorf("Footer mismatch: got %v, want %v", gotEmbed.Footer, tt.expectedEmbed.Footer)
				}

				// Compare components
				if !reflect.DeepEqual(gotComponents, tt.expectedComponents) {
					t.Errorf("Components mismatch: got %v, want %v", gotComponents, tt.expectedComponents)
				}

				// Set timestamp back after tests
				gotEmbed.Timestamp = origTimestamp
			}
		})
	}
}
