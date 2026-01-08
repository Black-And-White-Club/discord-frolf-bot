package leaderboardhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

var testTagNumber = sharedtypes.TagNumber(5)

func TestLeaderboardHandlers_HandleTagAssignRequest(t *testing.T) {
	testMessageID := uuid.New()
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "valid_request",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`{"target_user_id":"user123","requestor_id":"req456","tag_number":5,"channel_id":"chan789","message_id":"` + testMessageID.String() + `"}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)
						*payload = sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
							TargetUserID: sharedtypes.DiscordID("user123"),
							RequestorID:  "req456",
							TagNumber:    testTagNumber,
							ChannelID:    "chan789",
							MessageID:    testMessageID.String(),
						}
						return nil
					}).Times(1)

				expectedPayload := leaderboardevents.LeaderboardTagAssignmentRequestedPayloadV1{
					UserID:     sharedtypes.DiscordID("user123"),
					UpdateID:   sharedtypes.RoundID(testMessageID),
					TagNumber:  &testTagNumber,
					Source:     "discord_claim",
					UpdateType: "manual_assignment",
				}

				mockMsg := message.NewMessage("test", nil)
				mockMsg.Metadata = message.Metadata{
					"user_id":      "user123",
					"requestor_id": "req456",
					"channel_id":   "chan789",
					"message_id":   testMessageID.String(),
				}

				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), expectedPayload, leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1).
					Return(mockMsg, nil).Times(1)
			},
		},
		{
			name: "missing_required_fields",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`{"target_user_id":"","requestor_id":"req456","tag_number":0}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)
						*payload = sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
							TargetUserID: sharedtypes.DiscordID(""),
							RequestorID:  "req456",
							TagNumber:    sharedtypes.TagNumber(0),
							ChannelID:    "",
							MessageID:    "",
						}
						return nil
					}).Times(1)
			},
		},
		{
			name: "empty_message_id",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`{"target_user_id":"user123","requestor_id":"req456","tag_number":5,"channel_id":"chan789","message_id":""}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)
						*payload = sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
							TargetUserID: sharedtypes.DiscordID("user123"),
							RequestorID:  "req456",
							TagNumber:    testTagNumber,
							ChannelID:    "chan789",
							MessageID:    "",
						}
						return nil
					}).Times(1)
			},
		},
		{
			name: "invalid_message_id_format",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`{"target_user_id":"user123","requestor_id":"req456","tag_number":5,"channel_id":"chan789","message_id":"invalid-uuid"}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)
						*payload = sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
							TargetUserID: sharedtypes.DiscordID("user123"),
							RequestorID:  "req456",
							TagNumber:    testTagNumber,
							ChannelID:    "chan789",
							MessageID:    "invalid-uuid",
						}
						return nil
					}).Times(1)
			},
		},
		{
			name: "create_message_error",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`{"target_user_id":"user123","requestor_id":"req456","tag_number":5,"channel_id":"chan789","message_id":"` + testMessageID.String() + `"}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1)
						*payload = sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
							TargetUserID: sharedtypes.DiscordID("user123"),
							RequestorID:  "req456",
							TagNumber:    testTagNumber,
							ChannelID:    "chan789",
							MessageID:    testMessageID.String(),
						}
						return nil
					}).Times(1)

				expectedPayload := leaderboardevents.LeaderboardTagAssignmentRequestedPayloadV1{
					UserID:     sharedtypes.DiscordID("user123"),
					UpdateID:   sharedtypes.RoundID(testMessageID),
					TagNumber:  &testTagNumber,
					Source:     "discord_claim",
					UpdateType: "manual_assignment",
				}

				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), expectedPayload, leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1).
					Return(nil, errors.New("failed to create message")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			if tt.setup != nil {
				tt.setup(ctrl, mockHelper)
			}

			h := &LeaderboardHandlers{
				Logger:             mockLogger,
				Helpers:            mockHelper,
				LeaderboardDiscord: nil,
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagAssignRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && got == nil {
				t.Errorf("expected messages but got nil")
			}
			if !tt.wantErr && len(got) == 0 {
				t.Errorf("expected at least 1 message but got zero")
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagAssignedResponse(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_translation",
			msg: &message.Message{
				UUID: "test-uuid",
				Metadata: message.Metadata{
					"correlation_id": "test-correlation",
					"user_id":        "user123",
					"requestor_id":   "req456",
					"channel_id":     "chan789",
					"message_id":     "msg012",
				},
				Payload: []byte(`{"user_id":"user123","tag_number":5}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*leaderboardevents.LeaderboardTagAssignedPayloadV1)
						*payload = leaderboardevents.LeaderboardTagAssignedPayloadV1{
							UserID:    "user123",
							TagNumber: &testTagNumber,
						}
						return nil
					}).Times(1)

				expectedPayload := sharedleaderboardevents.LeaderboardTagAssignedPayloadV1{
					TargetUserID: "user123",
					TagNumber:    5,
					ChannelID:    "chan789",
					MessageID:    "msg012",
				}

				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), expectedPayload, sharedleaderboardevents.LeaderboardTagAssignedV1).
					Return(message.NewMessage("test", nil), nil).Times(1)
			},
		},
		{
			name: "unmarshal_error",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`invalid`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("unmarshal error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			if tt.setup != nil {
				tt.setup(ctrl, mockHelper)
			}

			h := &LeaderboardHandlers{
				Logger:             mockLogger,
				Helpers:            mockHelper,
				LeaderboardDiscord: nil,
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagAssignedResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && got == nil {
				t.Errorf("expected messages but got nil")
			}
			if !tt.wantErr && len(got) == 0 {
				t.Errorf("expected at least 1 message but got zero")
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagAssignFailedResponse(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_translation",
			msg: &message.Message{
				UUID: "test-uuid",
				Metadata: message.Metadata{
					"correlation_id": "test-correlation",
					"user_id":        "user123",
					"requestor_id":   "req456",
					"channel_id":     "chan789",
					"message_id":     "msg012",
				},
				Payload: []byte(`{"user_id":"user123","tag_number":5,"reason":"conflict"}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1)
						*payload = leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1{
							UserID:    "user123",
							TagNumber: &testTagNumber,
							Reason:    "conflict",
						}
						return nil
					}).Times(1)

				expectedPayload := sharedleaderboardevents.LeaderboardTagAssignFailedPayloadV1{
					TargetUserID: "user123",
					TagNumber:    testTagNumber,
					Reason:       "conflict",
					ChannelID:    "chan789",
					MessageID:    "msg012",
				}

				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), expectedPayload, sharedleaderboardevents.LeaderboardTagAssignFailedV1).
					Return(message.NewMessage("test", nil), nil).Times(1)
			},
		},
		{
			name: "unmarshal_error",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
				Payload:  []byte(`invalid`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("unmarshal error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			if tt.setup != nil {
				tt.setup(ctrl, mockHelper)
			}

			h := &LeaderboardHandlers{
				Logger:             mockLogger,
				Helpers:            mockHelper,
				LeaderboardDiscord: nil,
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagAssignFailedResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignFailedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && got == nil {
				t.Errorf("expected messages but got nil")
			}
			if !tt.wantErr && len(got) == 0 {
				t.Errorf("expected at least 1 message but got zero")
			}
		})
	}
}
