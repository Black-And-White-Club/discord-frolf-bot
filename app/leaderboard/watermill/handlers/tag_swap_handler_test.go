package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"reflect"
	"testing"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagSwapRequest(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_tag_swap_request",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"user1_id": "user1", "user2_id": "user2", "requestor_id": "requestor", "channel_id": "channel", "message_id": "message"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want: []*message.Message{
				{
					Payload: []byte(`{"requestor_id":"requestor","target_id":"user2"}`),
					Metadata: message.Metadata{
						"user_id":    "requestor",
						"channel_id": "channel",
						"message_id": "message",
					},
				},
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				// 1. Ensure UnmarshalPayload uses the EXACT type from the handler
				expectedPayload := discordleaderboardevents.LeaderboardTagSwapRequestPayload{
					User1ID:     "user1",
					User2ID:     "user2",
					RequestorID: "requestor",
					ChannelID:   "channel",
					MessageID:   "message",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordleaderboardevents.LeaderboardTagSwapRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						// Type-assert to the specific pointer type
						target := v.(*discordleaderboardevents.LeaderboardTagSwapRequestPayload)
						*target = expectedPayload
						return nil
					}).
					Times(1)

				// 2. Match CreateResultMessage arguments exactly
				backendPayload := leaderboardevents.TagSwapRequestedPayload{
					RequestorID: sharedtypes.DiscordID("requestor"),
					TargetID:    sharedtypes.DiscordID("user2"),
				}

				payloadBytes, _ := json.Marshal(backendPayload)

				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Eq(backendPayload), // Use gomock.Eq for exact payload match
						leaderboardevents.TagSwapRequested,
					).
					Return(&message.Message{
						Payload:  payloadBytes,
						Metadata: make(message.Metadata),
					}, nil).
					Times(1)
			},
		},
		{
			name: "invalid_payload",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"user1_id": "", "user2_id": "user2", "requestor_id": "requestor", "channel_id": "channel", "message_id": "message"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordleaderboardevents.LeaderboardTagSwapRequestPayload{})).
					Return(nil).
					Times(1)
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockTracer := noop.NewTracerProvider().Tracer("test")
			mockMetrics := &discordmetrics.NoOpMetrics{}

			tt.setup(ctrl, mockHelper)

			h := &LeaderboardHandlers{
				Logger:  mockLogger,
				Helpers: mockHelper,
				Tracer:  mockTracer,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagSwapRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagSwapRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagSwappedResponse(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_tag_swapped_response",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"requestor_id": "requestor", "target_id": "target"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"user_id":        "user_id",
					"channel_id":     "channel_id",
					"message_id":     "message_id",
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := leaderboardevents.TagSwapProcessedPayload{
					RequestorID: "requestor",
					TargetID:    "target",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.TagSwapProcessedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*leaderboardevents.TagSwapProcessedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				discordPayload := discordleaderboardevents.LeaderboardTagSwappedPayload{
					User1ID:   "requestor",
					User2ID:   "target",
					ChannelID: "channel_id",
					MessageID: "message_id",
				}

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), discordPayload, discordleaderboardevents.LeaderboardTagSwappedTopic).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "missing_metadata",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"requestor_id": "requestor", "target_id": "target"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"user_id":        "",
					"channel_id":     "channel_id",
					"message_id":     "message_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.TagSwapProcessedPayload{})).
					Return(nil).
					Times(1)
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockTracer := noop.NewTracerProvider().Tracer("test")
			mockMetrics := &discordmetrics.NoOpMetrics{}

			tt.setup(ctrl, mockHelper)

			h := &LeaderboardHandlers{
				Logger:  mockLogger,
				Helpers: mockHelper,
				Tracer:  mockTracer,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagSwappedResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwappedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagSwappedResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagSwapFailedResponse(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_tag_swap_failed_response",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"requestor_id": "requestor", "target_id": "target", "reason": "reason"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"user_id":        "user_id",
					"channel_id":     "channel_id",
					"message_id":     "message_id",
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := leaderboardevents.TagSwapFailedPayload{
					RequestorID: "requestor",
					TargetID:    "target",
					Reason:      "reason",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.TagSwapFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*leaderboardevents.TagSwapFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				discordPayload := discordleaderboardevents.LeaderboardTagSwapFailedPayload{
					User1ID:   "requestor",
					User2ID:   "target",
					Reason:    "reason",
					ChannelID: "channel_id",
					MessageID: "message_id",
				}

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), discordPayload, discordleaderboardevents.LeaderboardTagSwapFailedTopic).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "missing_metadata",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"requestor_id": "requestor", "target_id": "target", "reason": "reason"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"user_id":        "",
					"channel_id":     "channel_id",
					"message_id":     "message_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&leaderboardevents.TagSwapFailedPayload{})).
					Return(nil).
					Times(1)
			},
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockTracer := noop.NewTracerProvider().Tracer("test")
			mockMetrics := &discordmetrics.NoOpMetrics{}

			tt.setup(ctrl, mockHelper)

			h := &LeaderboardHandlers{
				Logger:  mockLogger,
				Helpers: mockHelper,
				Tracer:  mockTracer,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleTagSwapFailedResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapFailedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagSwapFailedResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
