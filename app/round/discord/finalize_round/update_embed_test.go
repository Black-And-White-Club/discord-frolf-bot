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
		eventMessageID sharedtypes.RoundID
		expectErr      bool
	}{
		{
			name: "Successful finalization",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(&discordgo.User{Username: "TestUser"}, nil).
					AnyTimes()

				mockSession.EXPECT().
					GuildMember(gomock.Any(), gomock.Any()).
					Return(&discordgo.Member{Nick: ""}, nil).
					AnyTimes()

				mockSession.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					DoAndReturn(func(edit *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
						// Compare against the actual UUID string
						if edit.Channel != "test-channel" || edit.ID != testRoundID.String() {
							t.Errorf("Unexpected edit parameters: got %+v, want channel=%s, id=%s",
								edit, "test-channel", testRoundID.String())
						}

						return &discordgo.Message{
							ID:     edit.ID, // Use the same ID that was passed in
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
			eventMessageID: testRoundID,
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
			eventMessageID: testRoundID,
			expectErr:      true,
		},
		{
			name: "Transform fails due to User API error",
			setupMocks: func(mockSession *discordmocks.MockSession) {
				mockSession.EXPECT().
					User(gomock.Any()).
					Return(nil, fmt.Errorf("user not found")).
					AnyTimes()

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
			eventMessageID: testRoundID,
			expectErr:      false,
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
			eventMessageID: testRoundID,
			expectErr:      true,
		},
		{
			name: "Missing message ID",
			setupMocks: func(mockSession *discordmocks.MockSession) {
			},
			embedPayload: roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        testRoundID,
				Title:          "Test Round",
				Location:       (*roundtypes.Location)(strPtr("Test Course")),
				StartTime:      (*sharedtypes.StartTime)(timePtr(fixedTime)),
				EventMessageID: (*sharedtypes.RoundID)(&testRoundID),
			},
			channelID:      "test-channel",
			eventMessageID: sharedtypes.RoundID(uuid.Nil),
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
			eventMessageID: testRoundID,
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
					return fn(ctx)
				},
			}

			if tt.name == "Nil session" {
				frm.session = nil
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			// Call the function
			ctx := context.Background()
			got, err := frm.FinalizeScorecardEmbed(ctx, sharedtypes.RoundID(tt.eventMessageID), tt.channelID, tt.embedPayload)

			hasError := err != nil || got.Error != nil
			if hasError != tt.expectErr {
				t.Errorf("FinalizeScorecardEmbed() error expectation mismatch. Returned error: %v, Result error: %v, wantErr: %v", err, got.Error, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if got.Error != nil {
					t.Errorf("FinalizeScorecardEmbed() = %v, expected success, got error in result: %v", got, got.Error)
				}

				if got.Success == nil {
					t.Errorf("FinalizeScorecardEmbed() Success field is nil, expected a value on success")
				}
			}
		})
	}
}
