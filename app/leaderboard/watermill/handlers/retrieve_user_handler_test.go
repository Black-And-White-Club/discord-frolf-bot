package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/events/leaderboard"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetTagByDiscordID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()
	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful get tag by discord ID",
			msg: func() *message.Message {
				payload := discordleaderboardevents.LeaderboardTagAvailabilityRequestPayload{
					UserID:    "123456789",
					ChannelID: "channel123",
					MessageID: "message123",
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling GetTagByDiscordID request",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Successfully translated GetTagByDiscordID request",
						gomock.Any(),
					)
			},
			expectedError: false,
			expectedResult: func() []*message.Message {
				h := &LeaderboardHandlers{
					Logger:    mockLogger,
					Session:   mockSession,
					Config:    mockConfig,
					EventUtil: mockEventUtil,
				}
				payload := leaderboardevents.GetTagByDiscordIDRequestPayload{
					DiscordID: "123456789",
				}
				msg, _ := h.createResultMessage(nil, payload, leaderboardevents.GetTagByDiscordIDRequest)
				msg.Metadata.Set("correlation_id", "correlation123")
				return []*message.Message{msg}
			}(),
		},
		{
			name: "invalid payload - unmarshal error",
			msg: func() *message.Message {
				// Invalid JSON payload
				msg := message.NewMessage("test-id", []byte(`{invalid json`))
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling GetTagByDiscordID request",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to unmarshal payload",
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError:  true,
			expectedResult: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			h := &LeaderboardHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}
			got, err := h.HandleGetTagByDiscordID(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleGetTagByDiscordID() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if tt.expectedResult == nil {
				if got != nil {
					t.Errorf("expected nil result but got messages")
				}
				return
			}
			if len(got) != len(tt.expectedResult) {
				t.Errorf("expected %d messages, got %d", len(tt.expectedResult), len(got))
				return
			}
			for i, expectedMsg := range tt.expectedResult {
				var gotPayload, expectedPayload map[string]interface{}
				if err := json.Unmarshal(got[i].Payload, &gotPayload); err != nil {
					t.Fatalf("failed to unmarshal got payload: %v", err)
				}
				if err := json.Unmarshal(expectedMsg.Payload, &expectedPayload); err != nil {
					t.Fatalf("failed to unmarshal expected payload: %v", err)
				}
				if !reflect.DeepEqual(gotPayload, expectedPayload) {
					t.Errorf("message %d payload mismatch:\ngot:  %v\nwant: %v", i, gotPayload, expectedPayload)
				}
				if !reflect.DeepEqual(got[i].Metadata, expectedMsg.Metadata) {
					t.Errorf("message %d metadata mismatch:\ngot:  %v\nwant: %v", i, got[i].Metadata, expectedMsg.Metadata)
				}
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByDiscordIDResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()
	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful get tag by discord ID response",
			msg: func() *message.Message {
				payload := leaderboardevents.GetTagByDiscordIDResponsePayload{
					TagNumber: 123,
					ChannelID: "channel123",
					MessageID: "message123",
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling GetTagByDiscordIDResponse",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Successfully translated GetTagByDiscordIDResponse",
						gomock.Any(),
					)
			},
			expectedError: false,
			expectedResult: func() []*message.Message {
				h := &LeaderboardHandlers{
					Logger:    mockLogger,
					Session:   mockSession,
					Config:    mockConfig,
					EventUtil: mockEventUtil,
				}
				payload := discordleaderboardevents.LeaderboardTagAvailabilityResponsePayload{
					TagNumber: 123,
					ChannelID: "channel123",
					MessageID: "message123",
				}
				msg, _ := h.createResultMessage(nil, payload, discordleaderboardevents.LeaderboardTagAvailabilityResponseTopic)
				msg.Metadata.Set("correlation_id", "correlation123")
				return []*message.Message{msg}
			}(),
		},
		{
			name: "invalid payload - unmarshal error",
			msg: func() *message.Message {
				// Invalid JSON payload
				msg := message.NewMessage("test-id", []byte(`{invalid json`))
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling GetTagByDiscordIDResponse",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to unmarshal payload",
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError:  true,
			expectedResult: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			h := &LeaderboardHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}
			got, err := h.HandleGetTagByDiscordIDResponse(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleGetTagByDiscordIDResponse() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if tt.expectedResult == nil {
				if got != nil {
					t.Errorf("expected nil result but got messages")
				}
				return
			}
			if len(got) != len(tt.expectedResult) {
				t.Errorf("expected %d messages, got %d", len(tt.expectedResult), len(got))
				return
			}
			for i, expectedMsg := range tt.expectedResult {
				var gotPayload, expectedPayload map[string]interface{}
				if err := json.Unmarshal(got[i].Payload, &gotPayload); err != nil {
					t.Fatalf("failed to unmarshal got payload: %v", err)
				}
				if err := json.Unmarshal(expectedMsg.Payload, &expectedPayload); err != nil {
					t.Fatalf("failed to unmarshal expected payload: %v", err)
				}
				if !reflect.DeepEqual(gotPayload, expectedPayload) {
					t.Errorf("message %d payload mismatch:\ngot:  %v\nwant: %v", i, gotPayload, expectedPayload)
				}
				if !reflect.DeepEqual(got[i].Metadata, expectedMsg.Metadata) {
					t.Errorf("message %d metadata mismatch:\ngot:  %v\nwant: %v", i, got[i].Metadata, expectedMsg.Metadata)
				}
			}
		})
	}
}
