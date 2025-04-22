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

func Test_finalizeRoundManager_FinalizeScorecardEmbed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	strPtr := func(s string) *string {
		return &s
	}

	intPtr := func(i sharedtypes.Score) *sharedtypes.Score {
		return &i
	}

	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		setupMocks     func(mockSession *discordmocks.MockSession)
		embedPayload   roundevents.RoundFinalizedEmbedUpdatePayload
		channelID      string
		eventMessageID string
		expectErr      bool // Indicates if *any* error is expected (either in return or in result struct)
	}{
		{
			name: "Successful finalization",
			setupMocks: func(mockSession *discordmocks.MockSession) {
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
						// Validate edit matches what we expect - compare against tt.eventMessageID
						if edit.Channel != "test-channel" || edit.ID != "test-message" { // Use hardcoded values from test case for comparison
							t.Errorf("Unexpected edit parameters: got %+v, want channel=%s, id=%s",
								edit, "test-channel", "test-message")
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
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  intPtr(72),
					},
				},
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message", // Use hardcoded value here to match mock expectation
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
				RoundID:        sharedtypes.RoundID(testRoundID),
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants:   []roundtypes.Participant{},
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message", // Use hardcoded value here
			expectErr:      true,
		},
		{
			name: "Transform fails due to User API error",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(nil, fmt.Errorf("user not found")).
					AnyTimes()

				// Still expect the message edit call, as TransformRoundToFinalizedScorecard handles the user fetch error internally
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(&discordgo.Message{ID: "test-message"}, nil).
					Times(1)
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:   sharedtypes.RoundID(testRoundID),
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundtypes.Participant{
					{
						UserID: sharedtypes.DiscordID("user1"),
						Score:  intPtr(72),
					},
				},
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message", // Use hardcoded value here
			expectErr:      false,          // Transform error is logged, but the outer function doesn't return an error
		},
		{
			name: "Missing channel ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        sharedtypes.RoundID(testRoundID),
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*sharedtypes.StartTime)(timePtr(fixedTime)),
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "",
			eventMessageID: "test-message", // Use hardcoded value here
			expectErr:      true,
		},
		{
			name: "Missing message ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        sharedtypes.RoundID(testRoundID),
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*sharedtypes.StartTime)(timePtr(fixedTime)),
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: "",
			expectErr:      true,
		},
		{
			name: "Nil session",
			setupMocks: func(mockSession *discordmocks.MockSession) {
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        sharedtypes.RoundID(testRoundID),
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*sharedtypes.StartTime)(timePtr(fixedTime)),
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: "test-message", // Use hardcoded value here
			expectErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			mockLogger := loggerfrolfbot.NoOpLogger
			mockConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "guild-id",
				},
			}

			frm := &finalizeRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (FinalizeRoundOperationResult, error)) (FinalizeRoundOperationResult, error) {
					// Call the actual logic function directly for testing
					return fn(ctx)
				},
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

			// Check error expectation: check both the returned error and the error in the result struct
			hasError := err != nil || got.Error != nil
			if hasError != tt.expectErr {
				t.Errorf("FinalizeScorecardEmbed() error expectation mismatch. Returned error: %v, Result error: %v, wantErr: %v", err, got.Error, tt.expectErr)
				return
			}

			// If no error is expected, verify that the operation succeeded
			if !tt.expectErr {
				// Check that the result struct does not contain an error
				if got.Error != nil {
					t.Errorf("FinalizeScorecardEmbed() = %v, expected success, got error in result: %v", got, got.Error)
				}

				// Check if Success contains something meaningful (assuming success means Success field is not nil)
				if got.Success == nil {
					t.Errorf("FinalizeScorecardEmbed() Success field is nil, expected a value on success")
				}
			}
		})
	}
}
