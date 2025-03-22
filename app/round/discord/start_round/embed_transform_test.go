package startround

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_startRoundManager_TransformRoundToScorecard(t *testing.T) {
	// Helper function to create a pointer to a string
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a pointer to an int
	intPtr := func(i int) *int {
		return &i
	}

	// Helper function to create a pointer to a time
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Create fixed time for testing
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	// Test cases
	tests := []struct {
		name               string
		setup              func()
		payload            *roundevents.DiscordRoundStartPayload
		expectedEmbed      *discordgo.MessageEmbed
		expectedComponents []discordgo.MessageComponent
		expectError        bool
	}{
		{
			name:  "Basic round with no participants",
			setup: func() {},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:      roundtypes.ID(123),
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*roundtypes.StartTime)(timePtr(fixedTime)),
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
							CustomID: "round_enter_score|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Enter Score",
								ID:   "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: "round_join_late|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Join Round LATE",
								ID:   "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with participants (no nicknames)",
			setup: func() {

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
					GuildMember("guild-123", "user-123").
					Return(&discordgo.Member{Nick: ""}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-123", "user-456").
					Return(&discordgo.Member{Nick: ""}, nil).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
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
							CustomID: "round_enter_score|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Enter Score",
								ID:   "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: "round_join_late|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Join Round LATE",
								ID:   "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with participants (with nicknames)",
			setup: func() {

				// Mock User calls
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{Username: "TestUser1"}, nil).
					Times(1)

				mockSession.EXPECT().
					User("user-456").
					Return(&discordgo.User{Username: "TestUser2"}, nil).
					Times(1)

				// Mock GuildMember calls - with nicknames
				mockSession.EXPECT().
					GuildMember("guild-123", "user-123").
					Return(&discordgo.Member{Nick: "NickUser1"}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-123", "user-456").
					Return(&discordgo.Member{Nick: "NickUser2"}, nil).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
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
						Name:   "🏌️ NickUser1",
						Value:  "Score: --",
						Inline: true,
					},
					{
						Name:   "🏌️ NickUser2",
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
							CustomID: "round_enter_score|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Enter Score",
								ID:   "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: "round_join_late|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Join Round LATE",
								ID:   "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "User fetch fails",
			setup: func() {

				// Mock User call failure
				mockSession.EXPECT().
					User("user-123").
					Return(nil, fmt.Errorf("failed to fetch user")).
					Times(1)

			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{
					{
						UserID:    "user-123",
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
							CustomID: "round_enter_score|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Enter Score",
								ID:   "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: "round_join_late|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Join Round LATE",
								ID:   "🦇",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Guild member fetch fails (should use username)",
			setup: func() {

				// Mock User call success
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{Username: "TestUser1"}, nil).
					Times(1)

				// Mock GuildMember call failure
				mockSession.EXPECT().
					GuildMember("guild-123", "user-123").
					Return(nil, fmt.Errorf("failed to fetch guild member")).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{
					{
						UserID:    "user-123",
						TagNumber: nil,
						Score:     intPtr(0),
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
						Value:  "Score: +0",
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
							CustomID: "round_enter_score|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Enter Score",
								ID:   "💰",
							},
						},
						discordgo.Button{
							Label:    "Join Round LATE",
							Style:    discordgo.SecondaryButton,
							CustomID: "round_join_late|round-123",
							Emoji: &discordgo.ComponentEmoji{
								Name: "Join Round LATE",
								ID:   "🦇",
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

			// Create manager with mocks
			srm := &startRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			// IMPORTANT: Call the setup function before executing the test
			if tt.setup != nil {
				tt.setup()
			}

			// Call the function
			gotEmbed, gotComponents, err := srm.TransformRoundToScorecard(tt.payload)

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
