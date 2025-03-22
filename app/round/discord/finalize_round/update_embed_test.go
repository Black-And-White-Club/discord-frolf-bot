package finalizeround

import (
	"context"
	"fmt"
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

func Test_finalizeRoundManager_FinalizeScorecardEmbed(t *testing.T) {
	// Helper function to create a pointer to a string
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a pointer to an int
	intPtr := func(i int) *int {
		return &i
	}

	// Helper function to create a pointer to a time.Time
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	// Create fixed time for testing
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	// Test cases
	tests := []struct {
		name           string
		setupMocks     func(mockSession *discordmocks.MockSession)
		embedPayload   roundevents.RoundFinalizedEmbedUpdatePayload
		channelID      string
		eventMessageID string
		expectErr      bool
	}{
		{
			name: "Successful finalization",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Expected messageEdit that will be sent to Discord
				expectedEdit := &discordgo.MessageEdit{
					Channel: "test-channel",
					ID:      "test-message",
				}

				// Mock User calls for TransformRoundToFinalizedScorecard
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(&discordgo.User{Username: "TestUser"}, nil).
					AnyTimes()

				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(&discordgo.Member{Nick: ""}, nil).
					AnyTimes()

				// Mock the channel message edit
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Validate edit matches what we expect
						if edit.Channel != expectedEdit.Channel || edit.ID != expectedEdit.ID {
							t.Errorf("Unexpected edit parameters: got %+v, want channel=%s, id=%s",
								edit, expectedEdit.Channel, expectedEdit.ID)
						}

						// Return a mock message with the embed
						return &discordgo.Message{
							ID:     "test-message",
							Embeds: *edit.Embeds,
						}, nil
					}).
					Times(1)
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID: "user1",
						Score:  intPtr(72),
					},
				},
			},
			channelID:      "test-channel",
			eventMessageID: "test-message",
			expectErr:      false,
		},
		{
			name: "Edit message fails",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(&discordgo.User{Username: "TestUser"}, nil).
					AnyTimes()

				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(&discordgo.Member{Nick: ""}, nil).
					AnyTimes()

				// Mock the channel message edit to fail
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(nil, fmt.Errorf("discord API error")).
					Times(1)
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        roundtypes.ID(123),
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants:   []roundtypes.Participant{},
				EventMessageID: (*roundtypes.EventMessageID)(strPtr("test-message")),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message",
			expectErr:      true,
		},
		{
			name: "Transform fails due to User API error",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock User calls to fail for TransformRoundToFinalizedScorecard
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(nil, fmt.Errorf("user not found")).
					AnyTimes()

				// Still expect the message edit call
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(&discordgo.Message{ID: "test-message"}, nil).
					Times(1)
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID: "user1",
						Score:  intPtr(72),
					},
				},
			},
			channelID:      "test-channel",
			eventMessageID: "test-message",
			expectErr:      false,
		},
		{
			name: "Missing channel ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// No mocks needed as the function should return early
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID:      "",
			eventMessageID: "test-message",
			expectErr:      true,
		},
		{
			name: "Missing message ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// No mocks needed as the function should return early
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID:      "test-channel",
			eventMessageID: "",
			expectErr:      true,
		},
		{
			name: "Nil session",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// No mocks needed as we'll set session to nil
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   roundtypes.ID(123),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*roundtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message",
			expectErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup controller and mocks
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := observability.NewNoOpLogger()
			mockConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "guild-id",
				},
			}

			// Create manager with mocks
			frm := &finalizeRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			// Handle the nil session case
			if tt.name == "Nil session" {
				frm.session = nil
			}

			// Setup the specific test case mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			// Call the function
			ctx := context.Background()
			got, err := frm.FinalizeScorecardEmbed(ctx, tt.eventMessageID, tt.channelID, tt.embedPayload)

			// Check error expectation
			if (err != nil) != tt.expectErr {
				t.Errorf("FinalizeScorecardEmbed() error = %v, wantErr %v", err, tt.expectErr)
				return
			}

			// For successful cases, verify that we got a message back
			if !tt.expectErr && got == nil {
				t.Errorf("FinalizeScorecardEmbed() = nil, expected a message")
			}
		})
	}
}
