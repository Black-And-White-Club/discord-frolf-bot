package startround

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

func Test_startRoundManager_UpdateRoundToScorecard(t *testing.T) {
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

	// Test cases
	tests := []struct {
		name       string
		setupMocks func(mockSession *discordmocks.MockSession)
		payload    *roundevents.DiscordRoundStartPayload
		channelID  string
		messageID  string
		expectErr  bool
	}{
		{
			name: "Successful update",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// The expected messageEdit that will be sent to Discord
				expectedEdit := &discordgo.MessageEdit{
					Channel: "test-channel",
					ID:      "test-message",
				}

				// Mock the channel message edit with proper signature
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any(), gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Validate edit matches what we expect
						if edit.Channel != expectedEdit.Channel || edit.ID != expectedEdit.ID {
							t.Errorf("Unexpected edit parameters: got %+v, want channel=%s, id=%s",
								edit, expectedEdit.Channel, expectedEdit.ID)
						}
						// Return a mock message
						return &discordgo.Message{ID: "test-message"}, nil
					}).
					Times(1)

				// Mock User calls (similar to TransformRoundToScorecard test)
				// This is needed because we can't replace TransformRoundToScorecard method
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(&discordgo.User{Username: "TestUser"}, nil).
					AnyTimes()

				// Mock GuildMember calls
				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(&discordgo.Member{Nick: ""}, nil).
					AnyTimes()
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:      roundtypes.ID(123),
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: false,
		},
		{
			name: "Edit message fails",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock User and GuildMember calls for TransformRoundToScorecard
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(&discordgo.User{Username: "TestUser"}, nil).
					AnyTimes()

				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(&discordgo.Member{Nick: ""}, nil).
					AnyTimes()

				// Mock the channel message edit to fail with proper signature
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("discord API error")).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayload{
				RoundID:      roundtypes.ID(123),
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*roundtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipant{},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: true,
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
			srm := &startRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
			}

			// Setup the specific test case mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			// Call the function
			ctx := context.Background()
			err := srm.UpdateRoundToScorecard(ctx, tt.channelID, tt.messageID, tt.payload)

			// Check error expectation
			if (err != nil) != tt.expectErr {
				t.Errorf("UpdateRoundToScorecard() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}
