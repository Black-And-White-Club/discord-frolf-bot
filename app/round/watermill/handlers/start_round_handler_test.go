package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	startround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/start_round"
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

func TestRoundHandlers_HandleRoundStarted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "channel123"
	testEventMessageID := testRoundID

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockRoundDiscordInterface, *mocks.MockStartRoundManager)
	}{
		{
			name: "successful_round_start",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"event_message_id": "` + testEventMessageID.String() + `",
					"discord_channel_id": "` + testChannelID + `",
					"round_id": "` + testRoundID.String() + `"
				}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				expectedPayload := roundevents.DiscordRoundStartPayload{
					EventMessageID:   testEventMessageID,
					DiscordChannelID: testChannelID,
					RoundID:          testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordRoundStartPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.DiscordRoundStartPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// Return a zero value of StartRoundOperationResult
				mockStartRoundManager.EXPECT().
					UpdateRoundToScorecard(gomock.Any(), testChannelID, testEventMessageID.String(), &expectedPayload).
					Return(startround.StartRoundOperationResult{}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "invalid_payload_type",
			msg: &message.Message{
				UUID:    "2",
				Payload: []byte(`{}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid payload type")).
					Times(1)
			},
		},
		{
			name: "missing_event_message_id",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
					"discord_channel_id": "` + testChannelID + `",
					"round_id": "` + testRoundID.String() + `"
				}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				expectedPayload := roundevents.DiscordRoundStartPayload{
					DiscordChannelID: testChannelID,
					RoundID:          testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordRoundStartPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.DiscordRoundStartPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "update_round_to_scorecard_error",
			msg: &message.Message{
				UUID: "4",
				Payload: []byte(`{
					"event_message_id": "` + testEventMessageID.String() + `",
					"discord_channel_id": "` + testChannelID + `",
					"round_id": "` + testRoundID.String() + `"
				}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				expectedPayload := roundevents.DiscordRoundStartPayload{
					EventMessageID:   testEventMessageID,
					DiscordChannelID: testChannelID,
					RoundID:          testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordRoundStartPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.DiscordRoundStartPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockStartRoundManager.EXPECT().
					UpdateRoundToScorecard(gomock.Any(), testChannelID, testEventMessageID.String(), &expectedPayload).
					Return(startround.StartRoundOperationResult{}, errors.New("update failed")).
					Times(1)
			},
		},
		{
			name: "create_trace_message_error",
			msg: &message.Message{
				UUID: "5",
				Payload: []byte(`{
					"event_message_id": "` + testEventMessageID.String() + `",
					"discord_channel_id": "` + testChannelID + `",
					"round_id": "` + testRoundID.String() + `"
				}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockStartRoundManager *mocks.MockStartRoundManager) {
				expectedPayload := roundevents.DiscordRoundStartPayload{
					EventMessageID:   testEventMessageID,
					DiscordChannelID: testChannelID,
					RoundID:          testRoundID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.DiscordRoundStartPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.DiscordRoundStartPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockStartRoundManager.EXPECT().
					UpdateRoundToScorecard(gomock.Any(), testChannelID, testEventMessageID.String(), &expectedPayload).
					Return(startround.StartRoundOperationResult{}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(nil, errors.New("trace creation failed")).
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
			mockStartRoundManager := mocks.NewMockStartRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockHelper, mockRoundDiscord, mockStartRoundManager)

			mockRoundDiscord.EXPECT().
				GetStartRoundManager().
				Return(mockStartRoundManager).
				AnyTimes()

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

			got, err := h.HandleRoundStarted(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundStarted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundHandlers.HandleRoundStarted() = %v, want %v", got, tt.want)
			}
		})
	}
}
