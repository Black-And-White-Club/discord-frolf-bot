package leaderboardhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetTagByDiscordID(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_request_translation",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				discordPayload := sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{UserID: "user123"}
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{})).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1)
						*payload = discordPayload
						return nil
					}).Times(1)

				backendPayload := leaderboardevents.SoloTagNumberRequestPayloadV1{UserID: sharedtypes.DiscordID("user123")}
				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), backendPayload, leaderboardevents.GetTagByUserIDRequestedV1).
					Return(&message.Message{}, nil).Times(1)
			},
		},
		{
			name: "unmarshal_error",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "create_result_message_error",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				discordPayload := sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{UserID: "user123"}
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1)
						*payload = discordPayload
						return nil
					}).Times(1)
				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create error")).Times(1)
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
				LeaderboardDiscord: nil, // Not used in this handler
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleGetTagByDiscordID(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByDiscordID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetTagByDiscordID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByDiscordIDResponse(t *testing.T) {
	testTagNumber := sharedtypes.TagNumber(123)
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_response_translation",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"topic": leaderboardevents.GetTagNumberResponseV1, "correlation_id": "test-correlation"},
				Payload:  []byte(`{"tag_number": 123}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				backendPayload := leaderboardevents.GetTagNumberResponsePayloadV1{TagNumber: &testTagNumber}
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.GetTagNumberResponsePayloadV1{})).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*leaderboardevents.GetTagNumberResponsePayloadV1)
						payload.TagNumber = backendPayload.TagNumber
						return nil
					}).Times(1)

				discordPayload := sharedleaderboardevents.LeaderboardTagAvailabilityResponsePayloadV1{TagNumber: 123}
				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), discordPayload, sharedleaderboardevents.LeaderboardTagAvailabilityResponseV1).
					Return(&message.Message{}, nil).Times(1)
			},
		},
		{
			name: "handle_failed_event",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"topic": leaderboardevents.GetTagNumberFailedV1, "correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: false,
			setup:   func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {},
		},
		{
			name: "unexpected_topic",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"topic": "unknown-topic", "correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup:   func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {},
		},
		{
			name: "unmarshal_error_in_response",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"topic": leaderboardevents.GetTagNumberResponseV1, "correlation_id": "test-correlation"},
				Payload:  []byte(`invalid`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "create_result_message_error_in_response",
			msg: &message.Message{
				UUID:     "test-uuid",
				Metadata: message.Metadata{"topic": leaderboardevents.GetTagNumberResponseV1, "correlation_id": "test-correlation"},
				Payload:  []byte(`{"tag_number": 123}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				backendPayload := leaderboardevents.GetTagNumberResponsePayloadV1{TagNumber: &testTagNumber}
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, v interface{}) error {
						payload := v.(*leaderboardevents.GetTagNumberResponsePayloadV1)
						payload.TagNumber = backendPayload.TagNumber
						return nil
					}).Times(1)
				mockHelper.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create error")).Times(1)
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
				LeaderboardDiscord: nil, // Not used in this handler
				Tracer:             mockTracer,
				Metrics:            mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleGetTagByDiscordIDResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByDiscordIDResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetTagByDiscordIDResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
