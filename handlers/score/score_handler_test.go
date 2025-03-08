package scorehandlers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordscoreevents "github.com/Black-And-White-Club/discord-frolf-bot/events/score"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleScoreUpdateRequest(t *testing.T) {
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
			name: "successful score update",
			msg: func() *message.Message {
				score := 72
				payload := discordscoreevents.ScoreUpdateRequestPayload{
					RoundID:     "round123",
					Participant: "player123",
					Score:       &score,
					TagNumber:   123,
					UserID:      "user123",
					ChannelID:   "channel123",
					MessageID:   "message123",
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("user_id", "user123")
				msg.Metadata.Set("channel_id", "channel123")
				msg.Metadata.Set("message_id", "message123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling ScoreUpdateRequest",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Successfully translated ScoreUpdateRequest",
						gomock.Any(),
					)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"domain": "score",
											"event_name": "score.update.request",
											"round_id": "round123",
											"participant": "player123",
											"score": 72,
											"tag_number": 123,
											"timestamp": "0001-01-01T00:00:00Z"
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "user123")
					msg.Metadata.Set("channel_id", "channel123")
					msg.Metadata.Set("message_id", "message123")
					msg.Metadata.Set("domain", "score")
					msg.Metadata.Set("topic", "score.update.request")
					msg.Metadata.Set("handler_name", "createResultMessage")
					return msg
				}(),
			},
		},
		{
			name: "invalid payload - missing required fields",
			msg: func() *message.Message {
				payload := discordscoreevents.ScoreUpdateRequestPayload{
					UserID:    "user123",
					ChannelID: "channel123",
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
						"Handling ScoreUpdateRequest",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Invalid ScoreUpdateRequest payload",
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
			h := &ScoreHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}
			got, err := h.HandleScoreUpdateRequest(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleScoreUpdateRequest() error = %v, expectedError %v", err, tt.expectedError)
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

func TestScoreHandlers_HandleScoreUpdateResponse(t *testing.T) {
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
			name: "successful score update response",
			msg: func() *message.Message {
				payload := scoreevents.ScoreUpdateResponsePayload{
					CommonMetadata: events.CommonMetadata{
						Domain:    "score",
						EventName: "score.update.response",
						Timestamp: time.Time{}, // zero time
					},
					Success:     true,
					RoundID:     "round123",
					Participant: "player123",
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("user_id", "user123")
				msg.Metadata.Set("channel_id", "channel123")
				msg.Metadata.Set("message_id", "message123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling ScoreUpdateResponse",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Successfully translated ScoreUpdateResponse",
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
									"domain": "score",
									"event_name": "score.update.response",
									"timestamp": "0001-01-01T00:00:00Z",
									"success": true,
									"round_id": "round123",
									"participant": "player123",
									"user_id": "user123",
									"channel_id": "channel123",
									"message_id": "message123"
							}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "user123")
					msg.Metadata.Set("channel_id", "channel123")
					msg.Metadata.Set("message_id", "message123")
					msg.Metadata.Set("domain", "score")
					msg.Metadata.Set("topic", "discord.score.update.response")
					msg.Metadata.Set("handler_name", "createResultMessage")
					return msg
				}(),
			},
		},
		{
			name: "missing required metadata",
			msg: func() *message.Message {
				payload := scoreevents.ScoreUpdateResponsePayload{
					Success:     true,
					RoundID:     "round123",
					Participant: "player123",
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				// Deliberately omitting user_id and channel_id
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(
						gomock.Any(),
						"Handling ScoreUpdateResponse",
						gomock.Any(),
					)
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Missing required metadata for ScoreUpdateResponse",
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
			h := &ScoreHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}
			got, err := h.HandleScoreUpdateResponse(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleScoreUpdateResponse() error = %v, expectedError %v", err, tt.expectedError)
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
