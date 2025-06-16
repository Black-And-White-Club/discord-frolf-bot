package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"testing"

	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
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

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "test-channel"
	testParticipant := sharedtypes.DiscordID("user123")
	testScore := sharedtypes.Score(+18)
	testEventMessageID := "event-msg-123"

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockScoreRoundManager)
	}{
		{
			name: "successful_score_update",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
									"round_id": "` + testRoundID.String() + `",
									"channel_id": "` + testChannelID + `",
									"participant": "` + string(testParticipant) + `",
									"score": ` + strconv.Itoa(int(testScore)) + `,
									"event_message_id": "` + testEventMessageID + `"
							}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil, // Handler returns nil, nil (no messages)
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.ParticipantScoreUpdatedPayload{
					RoundID:        testRoundID,
					ChannelID:      testChannelID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testEventMessageID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantScoreUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantScoreUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// Mock the UpdateScoreEmbed call that the handler actually makes
				mockScoreManager.EXPECT().
					UpdateScoreEmbed(
						gomock.Any(),
						testChannelID,
						testEventMessageID,
						testParticipant,
						&testScore,
					).Return(scoreround.ScoreRoundOperationResult{}, nil).Times(1)
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
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid payload")).
					Times(1)
			},
		},
		{
			name: "update_score_embed_fails",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
									"round_id": "` + testRoundID.String() + `",
									"channel_id": "` + testChannelID + `",
									"participant": "` + string(testParticipant) + `",
									"score": ` + strconv.Itoa(int(testScore)) + `,
									"event_message_id": "` + testEventMessageID + `"
							}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.ParticipantScoreUpdatedPayload{
					RoundID:        testRoundID,
					ChannelID:      testChannelID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testEventMessageID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantScoreUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantScoreUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockScoreManager.EXPECT().
					UpdateScoreEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(scoreround.ScoreRoundOperationResult{}, errors.New("update failed")).
					Times(1)
			},
		},
		{
			name: "update_score_embed_result_has_error",
			msg: &message.Message{
				UUID: "4",
				Payload: []byte(`{
									"round_id": "` + testRoundID.String() + `",
									"channel_id": "` + testChannelID + `",
									"participant": "` + string(testParticipant) + `",
									"score": ` + strconv.Itoa(int(testScore)) + `,
									"event_message_id": "` + testEventMessageID + `"
							}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.ParticipantScoreUpdatedPayload{
					RoundID:        testRoundID,
					ChannelID:      testChannelID,
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testEventMessageID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantScoreUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantScoreUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockScoreManager.EXPECT().
					UpdateScoreEmbed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(scoreround.ScoreRoundOperationResult{
						Error: errors.New("result error"),
					}, nil).
					Times(1)
			},
		},
		{
			name: "missing_channel_id_uses_config",
			msg: &message.Message{
				UUID: "5",
				Payload: []byte(`{
									"round_id": "` + testRoundID.String() + `",
									"channel_id": "",
									"participant": "` + string(testParticipant) + `",
									"score": ` + strconv.Itoa(int(testScore)) + `,
									"event_message_id": "` + testEventMessageID + `"
							}`),
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.ParticipantScoreUpdatedPayload{
					RoundID:        testRoundID,
					ChannelID:      "", // Empty in payload
					Participant:    testParticipant,
					Score:          testScore,
					EventMessageID: testEventMessageID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantScoreUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantScoreUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// Should use config channel ID when payload channel ID is empty
				mockScoreManager.EXPECT().
					UpdateScoreEmbed(
						gomock.Any(),
						"config-channel", // From config fallback
						testEventMessageID,
						testParticipant,
						&testScore,
					).Return(scoreround.ScoreRoundOperationResult{}, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockScoreManager := mocks.NewMockScoreRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockHelper, mockScoreManager)

			mockRoundDiscord.EXPECT().
				GetScoreRoundManager().
				Return(mockScoreManager).
				AnyTimes()

			h := &RoundHandlers{
				Logger: mockLogger,
				Config: &config.Config{
					Discord: config.DiscordConfig{
						EventChannelID: "config-channel", // Add config channel for fallback test
					},
				},
				Helpers:      mockHelper,
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleParticipantScoreUpdated(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleParticipantScoreUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateError(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("user123")
	testError := "database connection failed"

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockScoreRoundManager)
	}{
		{
			name: "successful_error_handling",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
									"score_update_request": {
											"round_id": "` + testRoundID.String() + `",
											"participant": "` + string(testParticipant) + `"
									},
									"error": "` + testError + `"
							}`),
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
					},
					Error: testError,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundScoreUpdateErrorPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundScoreUpdateErrorPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockScoreManager.EXPECT().
					SendScoreUpdateError(gomock.Any(), testParticipant, testError).
					Return(scoreround.ScoreRoundOperationResult{}, nil).
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
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid payload")).
					Times(1)
			},
		},
		{
			name: "empty_error_message",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
									"score_update_request": {
											"round_id": "` + testRoundID.String() + `",
											"participant": "` + string(testParticipant) + `"
									},
									"error": ""
							}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
					},
					Error: "", // Empty error
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundScoreUpdateErrorPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundScoreUpdateErrorPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "send_error_fails",
			msg: &message.Message{
				UUID: "4",
				Payload: []byte(`{
									"score_update_request": {
											"round_id": "` + testRoundID.String() + `",
											"participant": "` + string(testParticipant) + `"
									},
									"error": "` + testError + `"
							}`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockScoreManager *mocks.MockScoreRoundManager) {
				expectedPayload := roundevents.RoundScoreUpdateErrorPayload{
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     testRoundID,
						Participant: testParticipant,
					},
					Error: testError,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundScoreUpdateErrorPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundScoreUpdateErrorPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockScoreManager.EXPECT().
					SendScoreUpdateError(gomock.Any(), testParticipant, testError).
					Return(scoreround.ScoreRoundOperationResult{}, errors.New("send failed")).
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
			mockScoreManager := mocks.NewMockScoreRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockHelper, mockScoreManager)

			mockRoundDiscord.EXPECT().
				GetScoreRoundManager().
				Return(mockScoreManager).
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

			got, err := h.HandleScoreUpdateError(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleScoreUpdateError() = %v, want %v", got, tt.want)
			}
		})
	}
}
