package finalizeround

import (
	"context"
	"fmt"
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

func Test_finalizeRoundManager_TransformRoundToFinalizedScorecard(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	// Helper function to create a pointer to a string
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a pointer to an int
	intPtr := func(i sharedtypes.Score) *sharedtypes.Score {
		return &i
	}

	// Helper function to create a pointer to a time
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Create fixed time for testing
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	// Test cases
	tests := []struct {
		name               string
		setup              func(mockSession *discordmocks.MockSession)
		payload            *roundevents.RoundFinalizedEmbedUpdatePayload
		expectedEmbed      *discordgo.MessageEmbed
		expectedComponents []discordgo.MessageComponent
		expectError        bool
	}{
		{
			name: "Basic round with no participants",
			setup: func(mockSession *discordmocks.MockSession) {
				// No setup needed for this case
			},
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:      sharedtypes.RoundID(testRoundID),
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{}, // Use correct type
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at Test Course has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
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
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
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
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user-123"),
						TagNumber: nil,
						Score:     intPtr(2),
					},
					{
						UserID:    sharedtypes.DiscordID("user-456"),
						TagNumber: nil,
						Score:     intPtr(-1),
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at Test Course has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
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
						Name:   "🥇 TestUser2",
						Value:  "Score: -1 (<@user-456>)",
						Inline: true,
					},
					{
						Name:   "🗑️ TestUser1",
						Value:  "Score: +2 (<@user-123>)",
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with participants (with nicknames)",
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

				// Mock GuildMember calls - with nicknames
				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: "NickUser1"}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-456").
					Return(&discordgo.Member{Nick: "NickUser2"}, nil).
					Times(1)
			},
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user-123"),
						TagNumber: nil,
						Score:     intPtr(0),
					},
					{
						UserID:    sharedtypes.DiscordID("user-456"),
						TagNumber: nil,
						Score:     nil,
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at Test Course has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
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
						Name:   "🥇 NickUser1",
						Value:  "Score: Even (<@user-123>)",
						Inline: true,
					},
					{
						Name:   "🗑️ NickUser2",
						Value:  "Score: -- (<@user-456>)",
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "User fetch fails",
			setup: func(mockSession *discordmocks.MockSession) {
				// Mock User call failure
				mockSession.EXPECT().
					User("user-123").
					Return(nil, fmt.Errorf("failed to fetch user")).
					Times(1)
			},
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user-123"),
						TagNumber: nil,
						Score:     nil,
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at Test Course has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
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
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Guild member fetch fails (should use username)",
			setup: func(mockSession *discordmocks.MockSession) {
				// Mock User call success
				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{Username: "TestUser1"}, nil).
					Times(1)

				// Mock GuildMember call failure
				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(nil, fmt.Errorf("failed to fetch guild member")).
					Times(1)
			},
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("user-123"),
						TagNumber: nil,
						Score:     intPtr(-5),
					},
				},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at Test Course has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
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
						Name:   "😢 TestUser1",
						Value:  "Score: -5 (<@user-123>)",
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Round with null location",
			setup: func(mockSession *discordmocks.MockSession) {
				// No setup needed for this case
			},
			payload: &roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:      sharedtypes.RoundID(testRoundID),
				Title:        "Test Round",
				Location:     nil,
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{},
			},
			expectedEmbed: &discordgo.MessageEmbed{
				Title:       "**Test Round** - Round Finalized",
				Description: "Round at  has been finalized. Admin/Editor access required for score updates.",
				Color:       0x0000FF, // Blue for finalized round
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "📅 Started",
						Value: fmt.Sprintf("<t:%d:f>", fixedTime.Unix()),
					},
					{
						Name:  "📍 Location",
						Value: "",
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Round has been finalized. Only admins/editors can update scores.",
				},
				// Timestamp will be checked separately
			},
			expectedComponents: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Score Override",
							Style:    discordgo.DangerButton,
							CustomID: fmt.Sprintf("round_bulk_score_override|%s", testRoundID),
							Emoji:    &discordgo.ComponentEmoji{Name: "🛠️"},
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
			frm := &finalizeRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (FinalizeRoundOperationResult, error)) (FinalizeRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			// Call the function
			gotEmbed, gotComponents, err := frm.TransformRoundToFinalizedScorecard(*tt.payload)

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
				if gotEmbed.Title != tt.expectedEmbed.Title {
					t.Errorf("Title mismatch: got %v, want %v", gotEmbed.Title, tt.expectedEmbed.Title)
				}
				if gotEmbed.Description != tt.expectedEmbed.Description {
					t.Errorf("Description mismatch: got %v, want %v", gotEmbed.Description, tt.expectedEmbed.Description)
				}
				if gotEmbed.Color != tt.expectedEmbed.Color {
					t.Errorf("Color mismatch: got %v, want %v", gotEmbed.Color, tt.expectedEmbed.Color)
				}

				// Compare fields individually since reflect.DeepEqual compares pointers
				if len(gotEmbed.Fields) != len(tt.expectedEmbed.Fields) {
					t.Errorf("Fields length mismatch: got %d, want %d", len(gotEmbed.Fields), len(tt.expectedEmbed.Fields))
				} else {
					for i, gotField := range gotEmbed.Fields {
						expectedField := tt.expectedEmbed.Fields[i]
						if gotField.Name != expectedField.Name {
							t.Errorf("Field %d name mismatch: got %v, want %v", i, gotField.Name, expectedField.Name)
						}
						if gotField.Value != expectedField.Value {
							t.Errorf("Field %d value mismatch: got %v, want %v", i, gotField.Value, expectedField.Value)
						}
						if gotField.Inline != expectedField.Inline {
							t.Errorf("Field %d inline mismatch: got %v, want %v", i, gotField.Inline, expectedField.Inline)
						}
					}
				}
				if (gotEmbed.Footer == nil) != (tt.expectedEmbed.Footer == nil) {
					t.Errorf("Footer presence mismatch: got %v want %v", gotEmbed.Footer, tt.expectedEmbed.Footer)
				} else if gotEmbed.Footer != nil && tt.expectedEmbed.Footer != nil {
					if gotEmbed.Footer.Text != tt.expectedEmbed.Footer.Text {
						t.Errorf("Footer text mismatch: got %v want %v", gotEmbed.Footer.Text, tt.expectedEmbed.Footer.Text)
					}
				}

				// Compare components
				// Component logical comparison (label, style, custom ID, emoji name)
				if len(gotComponents) != len(tt.expectedComponents) {
					// lengths differ
					if len(gotComponents) == 0 && len(tt.expectedComponents) == 0 {
						// both empty, fine
					} else {
						t.Errorf("Components length mismatch: got %d want %d", len(gotComponents), len(tt.expectedComponents))
					}
				} else {
					for ci := range gotComponents {
						gotRow, ok1 := gotComponents[ci].(discordgo.ActionsRow)
						wantRow, ok2 := tt.expectedComponents[ci].(discordgo.ActionsRow)
						if !ok1 || !ok2 {
							// Try pointer form
							if pr, okp := gotComponents[ci].(*discordgo.ActionsRow); okp {
								gotRow = *pr
								ok1 = true
							}
							if pr, okp := tt.expectedComponents[ci].(*discordgo.ActionsRow); okp {
								wantRow = *pr
								ok2 = true
							}
						}
						if !ok1 || !ok2 {
							continue // skip unknown component types for now
						}
						if len(gotRow.Components) != len(wantRow.Components) {
							t.Errorf("Row %d components length mismatch: got %d want %d", ci, len(gotRow.Components), len(wantRow.Components))
							continue
						}
						for bi := range gotRow.Components {
							gb, gOK := gotRow.Components[bi].(discordgo.Button)
							if !gOK {
								if pb, okp := gotRow.Components[bi].(*discordgo.Button); okp {
									gb = *pb
									gOK = true
								}
							}
							wb, wOK := wantRow.Components[bi].(discordgo.Button)
							if !wOK {
								if pb, okp := wantRow.Components[bi].(*discordgo.Button); okp {
									wb = *pb
									wOK = true
								}
							}
							if !gOK || !wOK {
								continue
							}
							if gb.Label != wb.Label || gb.Style != wb.Style || gb.CustomID != wb.CustomID {
								t.Errorf("Button mismatch at row %d index %d: got {Label:%s Style:%d ID:%s} want {Label:%s Style:%d ID:%s}", ci, bi, gb.Label, gb.Style, gb.CustomID, wb.Label, wb.Style, wb.CustomID)
							}
							gEmoji := ""
							wEmoji := ""
							if gb.Emoji != nil {
								gEmoji = gb.Emoji.Name
							}
							if wb.Emoji != nil {
								wEmoji = wb.Emoji.Name
							}
							if gEmoji != wEmoji {
								t.Errorf("Emoji mismatch at row %d index %d: got %s want %s", ci, bi, gEmoji, wEmoji)
							}
						}
					}
				}

				// Set timestamp back after tests
				gotEmbed.Timestamp = origTimestamp
			}
		})
	}
}
