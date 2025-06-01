package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
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

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	eventMessageID := sharedtypes.RoundID(uuid.New())
	discordMessageID := "discord-msg-123"

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers, *mocks.MockFinalizeRoundManager)
	}{
		{
			name: "successful_round_finalized",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "discord_channel_id": "1234", "event_message_id": "` + eventMessageID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": discordMessageID, // Add the required metadata
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				expectedPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundFinalizedEmbedUpdatePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundFinalizedEmbedUpdatePayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				// Fix the function signature to match what the handler actually calls
				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, "1234", expectedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "fail_to_finalize_embed",
			msg: &message.Message{
				UUID:    "2",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "discord_channel_id": "1234", "event_message_id": "` + eventMessageID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": discordMessageID,
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				expectedPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundFinalizedEmbedUpdatePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundFinalizedEmbedUpdatePayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				// Fix the function signature and return an error
				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, "1234", expectedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{}, errors.New("failed to finalize embed")).
					Times(1)
			},
		},
		{
			name: "fail_create_trace_event",
			msg: &message.Message{
				UUID:    "3",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "discord_channel_id": "1234", "event_message_id": "` + eventMessageID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": discordMessageID,
				},
			},
			want:    []*message.Message{},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				expectedPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundFinalizedEmbedUpdatePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundFinalizedEmbedUpdatePayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				// Fix the function signature
				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, "1234", expectedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{}, nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(nil, errors.New("failed to create trace event")).
					Times(1)
			},
		},
		{
			name: "missing_discord_message_id_in_metadata",
			msg: &message.Message{
				UUID:    "4",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "discord_channel_id": "1234", "event_message_id": "` + eventMessageID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					// Missing discord_message_id
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				expectedPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundFinalizedEmbedUpdatePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundFinalizedEmbedUpdatePayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				// No other mocks needed since function should fail early
			},
		},
		{
			name: "finalize_result_has_error",
			msg: &message.Message{
				UUID:    "5",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "discord_channel_id": "1234", "event_message_id": "` + eventMessageID.String() + `"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": discordMessageID,
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				expectedPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundFinalizedEmbedUpdatePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundFinalizedEmbedUpdatePayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				// Return a result with an error in the result object
				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, "1234", expectedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{
						Error: errors.New("operation error in result"),
					}, nil).
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
			mockFinalizeRoundManager := mocks.NewMockFinalizeRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper, mockFinalizeRoundManager)

			h := &RoundHandlers{
				Logger: mockLogger,
				Config: &config.Config{
					Discord: config.DiscordConfig{
						ChannelID: "1234", // Add the channel ID to config
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

			got, err := h.HandleRoundFinalized(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundFinalized() = %v, want %v", got, tt.want)
			}
		})
	}
}
