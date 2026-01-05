package startround

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

func Test_startRoundManager_UpdateRoundToScorecard(t *testing.T) {
	// Helper function to create a pointer to a string
	strPtr := func(s string) *string {
		return &s
	}

	// Helper function to create a pointer to a time
	timePtr := func(t time.Time) *time.Time {
		return &t
	}

	testRoundID := sharedtypes.RoundID(uuid.New())
	// Create fixed time for testing
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	// Test cases
	tests := []struct {
		name       string
		setupMocks func(mockSession *discordmocks.MockSession)
		payload    *roundevents.DiscordRoundStartPayloadV1
		channelID  string
		messageID  string
		expectErr  bool
	}{
		{
			name: "Successful update",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock the channel message fetch
				mockSession.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(&discordgo.Message{
						ID:     "test-message",
						Embeds: []*discordgo.MessageEmbed{},
					}, nil).
					Times(1)

				// Mock the channel message edit
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Validate edit matches what we expect
						if edit.Channel != "test-channel" || edit.ID != "test-message" {
							t.Errorf("Unexpected edit parameters: got %+v, want channel=%s, id=%s",
								edit, "test-channel", "test-message")
						}
						// Return a mock message
						return &discordgo.Message{ID: "test-message"}, nil
					}).
					Times(1)

				// Mock User calls
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
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: false,
		},
		{
			name: "Edit message fails",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock the channel message fetch
				mockSession.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(&discordgo.Message{
						ID:     "test-message",
						Embeds: []*discordgo.MessageEmbed{},
					}, nil).
					Times(1)

				// Mock User and GuildMember calls
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
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name: "TransformRoundToScorecard fails",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock the channel message fetch
				mockSession.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(&discordgo.Message{
						ID:     "test-message",
						Embeds: []*discordgo.MessageEmbed{},
					}, nil).
					Times(1)

				// Mock User call to fail, but note the method continues despite this
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(nil, fmt.Errorf("user not found")).
					AnyTimes()

				// Mock GuildMember call to fail
				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("guild member not found")).
					AnyTimes()

				// Because TransformRoundToScorecard continues despite User failures,
				// ChannelMessageEditComplex WILL be called, so mock it to succeed
				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(&discordgo.Message{ID: "test-message"}, nil).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (*roundtypes.Location)(strPtr("Test Course")),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{
					{
						UserID: "test-user-1",
					},
				},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: false, // Changed to false since the operation will succeed
		},
		{
			name: "Fetch existing message fails",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// Mock the channel message fetch to fail
				mockSession.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(nil, fmt.Errorf("message not found")).
					Times(1)
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{},
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name: "Missing channel ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// No mocks needed, as the function should fail before making any API calls
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{},
			},
			channelID: "",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name: "Missing message ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				// No mocks needed, as the function should fail before making any API calls
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:      testRoundID,
				Title:        "Test Round",
				Location:     (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:    (*sharedtypes.StartTime)(timePtr(fixedTime)),
				Participants: []roundevents.RoundParticipantV1{},
			},
			channelID: "test-channel",
			messageID: "",
			expectErr: true,
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

			// Create manager with mocks
			srm := &startRoundManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error) {
					// Call the actual logic function directly for testing
					return fn(ctx)
				},
			}

			// Setup the specific test case mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			// Call the function
			ctx := context.Background()
			got, err := srm.UpdateRoundToScorecard(ctx, tt.channelID, tt.messageID, tt.payload)

			// Check error expectation
			hasError := err != nil || got.Error != nil
			if hasError != tt.expectErr {
				t.Errorf("UpdateRoundToScorecard() error expectation mismatch. Returned error: %v, Result error: %v, wantErr: %v", err, got.Error, tt.expectErr)
				return
			}

			// If no error is expected, verify that the operation succeeded
			if !tt.expectErr {
				// Check that the result struct does not contain an error
				if got.Error != nil {
					t.Errorf("UpdateRoundToScorecard() = %v, expected success, got error in result: %v", got, got.Error)
				}

				// Check if Success contains something meaningful
				if got.Success == nil {
					t.Errorf("UpdateRoundToScorecard() Success field is nil, expected a value on success")
				}
			}
		})
	}
}
