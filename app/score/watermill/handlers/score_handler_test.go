package scorehandlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	sharedscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_ScoreHandlers_HandleScoreUpdateRequest(t *testing.T) {
	validRoundID := uuid.New()

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful score update request",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "user_id": "12345", "score": 72}`))
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("backend-msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "user_id": "12345", "score": 72}`))
					msg.Metadata.Set("user_id", "12345")
					msg.Metadata.Set("channel_id", "channel-123")
					msg.Metadata.Set("message_id", "message-456")
					return msg
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{
					RoundID:   sharedtypes.RoundID(validRoundID),
					UserID:    "12345",
					Score:     72,
					ChannelID: "channel-123",
					MessageID: "message-456",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1) = *expectedPayload
						return nil
					})

				backendPayload := scoreevents.ScoreUpdateRequestedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					UserID:  "12345",
					Score:   72,
				}

				backendMsg := message.NewMessage("backend-msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "user_id": "12345", "score": 72}`))

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Eq(backendPayload), gomock.Eq(scoreevents.ScoreUpdateRequestedV1)).
					Return(backendMsg, nil)
			},
		},
		{
			name: "missing required fields in payload",
			msg: func() *message.Message {
				return message.NewMessage("msg-id", []byte(`{"user_id": "12345", "score": 72}`)) // Missing round_id
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{
					UserID: "12345",
					Score:  72,
					// Missing RoundID
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1) = *expectedPayload
						return nil
					})
			},
		},
		{
			name:    "failed to unmarshal payload",
			msg:     message.NewMessage("msg-id", []byte(`invalid payload`)),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.Any()).
					Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to create backend message",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "user_id": "12345", "score": 72}`))
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{
					RoundID:   sharedtypes.RoundID(validRoundID),
					UserID:    "12345",
					Score:     72,
					ChannelID: "channel-123",
					MessageID: "message-456",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*sharedscoreevents.ScoreUpdateRequestDiscordPayloadV1) = *expectedPayload
						return nil
					})

				backendPayload := scoreevents.ScoreUpdateRequestedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					UserID:  "12345",
					Score:   72,
				}

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Eq(backendPayload), gomock.Eq(scoreevents.ScoreUpdateRequestedV1)).
					Return(nil, errors.New("failed to create message"))
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
				tt.setup(ctrl, mockHelper, tt.msg)
			}

			h := &ScoreHandlers{
				Logger:  mockLogger,
				Helper:  mockHelper,
				Tracer:  mockTracer,
				Metrics: mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					var factory func() interface{}
					if unmarshalTo != nil {
						if fn, ok := unmarshalTo.(func() interface{}); ok {
							factory = fn
						} else {
							factory = func() interface{} { return utils.NewInstance(unmarshalTo) }
						}
					}
					return wrapHandler(handlerName, factory, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleScoreUpdateRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) == 0 && len(got) == 0 {
				return // no messages expected or returned — ✅
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unexpected number of messages: got %d, want %d", len(got), len(tt.want))
			}

			if len(got) > 0 && len(tt.want) > 0 {
				if !bytes.Equal(got[0].Payload, tt.want[0].Payload) {
					t.Errorf("Payload mismatch.\nGot:  %s\nWant: %s", got[0].Payload, tt.want[0].Payload)
				}

				if diff := cmp.Diff(got[0].Metadata, tt.want[0].Metadata); diff != "" {
					t.Errorf("Metadata mismatch (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func Test_ScoreHandlers_HandleScoreUpdateSuccess(t *testing.T) {
	validRoundID := uuid.New()

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful score update success handler",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "score": 72}`))
				msg.Metadata.Set("user_id", "12345")
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want: []*message.Message{
				func() *message.Message {
					return message.NewMessage("discord-msg-id", []byte(`{"type":"score_update_success","user_id":"12345","round_id":"`+validRoundID.String()+`","score":72,"message_id":"message-456"}`))
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdatedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Score:   72,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdatedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdatedPayloadV1) = *expectedPayload
						return nil
					})

				expectedResp := map[string]interface{}{
					"type":       "score_update_success",
					"user_id":    "12345",
					"round_id":   expectedPayload.RoundID,
					"score":      expectedPayload.Score,
					"message_id": "message-456",
				}

				discordMsg := message.NewMessage("discord-msg-id", []byte(`{"type":"score_update_success","user_id":"12345","round_id":"`+validRoundID.String()+`","score":72,"message_id":"message-456"}`))

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Eq(expectedResp), gomock.Eq(sharedscoreevents.ScoreUpdateResponseDiscordV1)).
					Return(discordMsg, nil)
			},
		},
		{
			name: "missing metadata for routing",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "score": 72}`))
				// Missing user_id metadata
				msg.Metadata.Set("channel_id", "channel-123")
				return msg
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdatedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Score:   72,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdatedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdatedPayloadV1) = *expectedPayload
						return nil
					})
			},
		},
		{
			name:    "failed to unmarshal payload",
			msg:     message.NewMessage("msg-id", []byte(`invalid payload`)),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.Any()).
					Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to create discord message",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "score": 72}`))
				msg.Metadata.Set("user_id", "12345")
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdatedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Score:   72,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdatedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdatedPayloadV1) = *expectedPayload
						return nil
					})

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Any(), gomock.Eq(sharedscoreevents.ScoreUpdateResponseDiscordV1)).
					Return(nil, errors.New("failed to create message"))
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
				tt.setup(ctrl, mockHelper, tt.msg)
			}

			h := &ScoreHandlers{
				Logger:  mockLogger,
				Helper:  mockHelper,
				Tracer:  mockTracer,
				Metrics: mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					var factory func() interface{}
					if unmarshalTo != nil {
						if fn, ok := unmarshalTo.(func() interface{}); ok {
							factory = fn
						} else {
							factory = func() interface{} { return utils.NewInstance(unmarshalTo) }
						}
					}
					return wrapHandler(handlerName, factory, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleScoreUpdateSuccess(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateSuccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) == 0 && len(got) == 0 {
				return // no messages expected or returned — ✅
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unexpected number of messages: got %d, want %d", len(got), len(tt.want))
			}

			if len(got) > 0 && len(tt.want) > 0 {
				if !bytes.Equal(got[0].Payload, tt.want[0].Payload) {
					t.Errorf("Payload mismatch.\nGot:  %s\nWant: %s", got[0].Payload, tt.want[0].Payload)
				}
			}
		})
	}
}

func Test_ScoreHandlers_HandleScoreUpdateFailure(t *testing.T) {
	validRoundID := uuid.New()

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful score update failure handler",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "error": "Score already recorded"}`))
				msg.Metadata.Set("user_id", "12345")
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want: []*message.Message{
				func() *message.Message {
					return message.NewMessage("discord-msg-id", []byte(`{"type":"score_update_failure","user_id":"12345","round_id":"`+validRoundID.String()+`","error":"Score already recorded","message_id":"message-456"}`))
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Reason:  "Score already recorded",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdateFailedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdateFailedPayloadV1) = *expectedPayload
						return nil
					})

				expectedResp := map[string]interface{}{
					"type":       "score_update_failure",
					"user_id":    "12345",
					"round_id":   expectedPayload.RoundID,
					"error":      expectedPayload.Reason,
					"message_id": "message-456",
				}

				discordMsg := message.NewMessage("discord-msg-id", []byte(`{"type":"score_update_failure","user_id":"12345","round_id":"`+validRoundID.String()+`","error":"Score already recorded","message_id":"message-456"}`))

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Eq(expectedResp), gomock.Eq(sharedscoreevents.ScoreUpdateFailedDiscordV1)).
					Return(discordMsg, nil)
			},
		},
		{
			name: "missing metadata for routing",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "error": "Score already recorded"}`))
				// Missing user_id and channel_id metadata
				return msg
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Reason:  "Score already recorded",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdateFailedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdateFailedPayloadV1) = *expectedPayload
						return nil
					})
			},
		},
		{
			name:    "failed to unmarshal payload",
			msg:     message.NewMessage("msg-id", []byte(`invalid payload`)),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.Any()).
					Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to create discord message",
			msg: func() *message.Message {
				msg := message.NewMessage("msg-id", []byte(`{"round_id": "`+validRoundID.String()+`", "error": "Score already recorded"}`))
				msg.Metadata.Set("user_id", "12345")
				msg.Metadata.Set("channel_id", "channel-123")
				msg.Metadata.Set("message_id", "message-456")
				return msg
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					RoundID: sharedtypes.RoundID(validRoundID),
					Reason:  "Score already recorded",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&scoreevents.ScoreUpdateFailedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*scoreevents.ScoreUpdateFailedPayloadV1) = *expectedPayload
						return nil
					})

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.Any(), gomock.Eq(sharedscoreevents.ScoreUpdateFailedDiscordV1)).
					Return(nil, errors.New("failed to create message"))
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
				tt.setup(ctrl, mockHelper, tt.msg)
			}

			h := &ScoreHandlers{
				Logger:  mockLogger,
				Helper:  mockHelper,
				Tracer:  mockTracer,
				Metrics: mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					var factory func() interface{}
					if unmarshalTo != nil {
						if fn, ok := unmarshalTo.(func() interface{}); ok {
							factory = fn
						} else {
							factory = func() interface{} { return utils.NewInstance(unmarshalTo) }
						}
					}
					return wrapHandler(handlerName, factory, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleScoreUpdateFailure(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateFailure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) == 0 && len(got) == 0 {
				return // no messages expected or returned — ✅
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unexpected number of messages: got %d, want %d", len(got), len(tt.want))
			}

			if len(got) > 0 && len(tt.want) > 0 {
				if !bytes.Equal(got[0].Payload, tt.want[0].Payload) {
					t.Errorf("Payload mismatch.\nGot:  %s\nWant: %s", got[0].Payload, tt.want[0].Payload)
				}
			}
		})
	}
}
