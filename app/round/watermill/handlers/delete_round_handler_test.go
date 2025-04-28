package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
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

func TestRoundHandlers_HandleRoundDeleted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers, *mocks.MockDeleteRoundManager)
	}{
		{
			name: "successful_round_deletion",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "` + testRoundID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testRoundID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: true}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "delete_round_event_embed_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "` + testRoundID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testRoundID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{}, errors.New("failed to delete round event embed")).
					Times(1)
			},
		},
		{
			name: "delete_round_event_embed_returns_false",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "` + testRoundID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testRoundID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: false}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "create_result_message_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "` + testRoundID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testRoundID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: true}, nil).
					Times(1)

				// This is where we simulate the failure
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(nil, errors.New("failed to create result message")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockDeleteRoundManager := mocks.NewMockDeleteRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper, mockDeleteRoundManager)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       &config.Config{}, // Provide a non-nil config
				Helpers:      mockHelper,
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleRoundDeleted(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}
