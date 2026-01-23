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
	timePtr := func(t time.Time) *time.Time { return &t }

	testRoundID := sharedtypes.RoundID(uuid.New())
	fixedTime := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setupMocks func(*discordmocks.MockSession)
		payload    *roundevents.DiscordRoundStartPayloadV1
		channelID  string
		messageID  string
		expectErr  bool
	}{
		{
			name: "Successful update",
			setupMocks: func(m *discordmocks.MockSession) {
				m.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(&discordgo.Message{ID: "test-message"}, nil)
				m.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(&discordgo.Message{ID: "test-message"}, nil)
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: false,
		},
		{
			name: "Edit message fails",
			setupMocks: func(m *discordmocks.MockSession) {
				m.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(&discordgo.Message{ID: "test-message"}, nil)
				m.EXPECT().
					ChannelMessageEditComplex(gomock.Any()).
					Return(nil, fmt.Errorf("discord API error"))
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name: "Fetch existing message fails",
			setupMocks: func(m *discordmocks.MockSession) {
				m.EXPECT().
					ChannelMessage("test-channel", "test-message").
					Return(nil, fmt.Errorf("message not found"))
			},
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID: "test-channel",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name:       "Missing channel ID",
			setupMocks: nil,
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID: "",
			messageID: "test-message",
			expectErr: true,
		},
		{
			name:       "Missing message ID",
			setupMocks: nil,
			payload: &roundevents.DiscordRoundStartPayloadV1{
				RoundID:   testRoundID,
				Title:     "Test Round",
				Location:  (roundtypes.Location)("Test Course"),
				StartTime: (*sharedtypes.StartTime)(timePtr(fixedTime)),
			},
			channelID: "test-channel",
			messageID: "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSession := discordmocks.NewMockSession(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockSession)
			}

			srm := &startRoundManager{
				session: mockSession,
				logger:  loggerfrolfbot.NoOpLogger,
				config:  &config.Config{},
				operationWrapper: func(ctx context.Context, name string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error) {
					return fn(ctx)
				},
			}

			got, err := srm.UpdateRoundToScorecard(context.Background(), tt.channelID, tt.messageID, tt.payload)

			hasError := err != nil || got.Error != nil
			if hasError != tt.expectErr {
				t.Errorf("UpdateRoundToScorecard() error = %v, result.Error = %v, wantErr %v", err, got.Error, tt.expectErr)
			}

			if !tt.expectErr && got.Success == nil {
				t.Errorf("Expected Success, got nil")
			}
		})
	}
}
