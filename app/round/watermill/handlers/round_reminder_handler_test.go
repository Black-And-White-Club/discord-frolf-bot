package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	roundreminder "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_reminder"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testReminderType := "start"
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockRoundDiscordInterface, *mocks.MockRoundReminderManager)
	}{
		{
			name: "successful_reminder_sent",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
				"round_id": "` + testRoundID.String() + `",
				"reminder_type": "` + testReminderType + `"
			}`),
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    []*message.Message{{UUID: "trace-msg"}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordReminderPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						payload := v.(*roundevents.DiscordReminderPayload)
						payload.RoundID = testRoundID
						payload.ReminderType = testReminderType
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					&roundevents.DiscordReminderPayload{
						RoundID:      testRoundID,
						ReminderType: testReminderType,
					},
				).Return(roundreminder.RoundReminderOperationResult{
					Success: true,
				}, nil).Times(1)
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					map[string]interface{}{
						"round_id":      testRoundID,
						"reminder_type": testReminderType,
						"status":        "reminder_sent",
					},
					roundevents.RoundTraceEvent,
				).Return(&message.Message{UUID: "trace-msg"}, nil).Times(1)
			},
		},
		{
			name: "reminder_send_error",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
				"round_id": "` + testRoundID.String() + `",
				"reminder_type": "` + testReminderType + `"
			}`),
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordReminderPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						payload := v.(*roundevents.DiscordReminderPayload)
						payload.RoundID = testRoundID
						payload.ReminderType = testReminderType
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundreminder.RoundReminderOperationResult{},
					errors.New("send error"),
				).Times(1)
			},
		},
		{
			name: "non_boolean_success_field",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
				"round_id": "` + testRoundID.String() + `",
				"reminder_type": "` + testReminderType + `"
			}`),
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    []*message.Message{{UUID: "trace-msg"}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordReminderPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						payload := v.(*roundevents.DiscordReminderPayload)
						payload.RoundID = testRoundID
						payload.ReminderType = testReminderType
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundreminder.RoundReminderOperationResult{
						Success: "true", // Non-boolean type
					},
					nil,
				).Times(1)
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					map[string]interface{}{
						"round_id":      testRoundID,
						"reminder_type": testReminderType,
						"status":        "reminder_failed",
					},
					roundevents.RoundTraceEvent,
				).Return(&message.Message{UUID: "trace-msg"}, nil).Times(1)
			},
		},
		{
			name: "invalid_payload_type",
			msg: &message.Message{
				UUID:     "4",
				Payload:  []byte(`{"invalid": "payload"}`),
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordReminderPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						return errors.New("invalid payload type")
					}).Times(1)
			},
		},
		{
			name: "create_trace_message_error",
			msg: &message.Message{
				UUID: "5",
				Payload: []byte(`{
				"round_id": "` + testRoundID.String() + `",
				"reminder_type": "` + testReminderType + `"
			}`),
				Metadata: message.Metadata{"correlation_id": "test-correlation"},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockReminderManager *mocks.MockRoundReminderManager) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordReminderPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						payload := v.(*roundevents.DiscordReminderPayload)
						payload.RoundID = testRoundID
						payload.ReminderType = testReminderType
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().GetRoundReminderManager().Return(mockReminderManager).Times(1)
				mockReminderManager.EXPECT().SendRoundReminder(
					gomock.Any(),
					gomock.Any(),
				).Return(
					roundreminder.RoundReminderOperationResult{
						Success: true,
					},
					nil,
				).Times(1)
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundTraceEvent,
				).Return(nil, errors.New("create error")).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockReminderManager := mocks.NewMockRoundReminderManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")
			tt.setup(ctrl, mockHelper, mockRoundDiscord, mockReminderManager)
			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{},
				Helpers:      mockHelper,
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}
			got, err := h.HandleRoundReminder(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundReminder() = %v, want %v", got, tt.want)
			}
		})
	}
}
