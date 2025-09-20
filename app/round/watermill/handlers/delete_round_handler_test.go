package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
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

// compareMessages compares two slices of messages by comparing their content instead of pointers
func compareMessages(got, want []*message.Message) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] == nil && want[i] == nil {
			continue
		}
		if got[i] == nil || want[i] == nil {
			return false
		}

		// Compare UUID, payload, and metadata
		if got[i].UUID != want[i].UUID {
			return false
		}

		if string(got[i].Payload) != string(want[i].Payload) {
			return false
		}

		// Compare metadata
		if len(got[i].Metadata) != len(want[i].Metadata) {
			return false
		}

		for key, value := range got[i].Metadata {
			if want[i].Metadata[key] != value {
				return false
			}
		}
	}

	return true
}

func TestRoundHandlers_HandleRoundDeleted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	// Define a test Discord message ID
	testDiscordMessageID := "123456789012345678" // Example Discord message ID string

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
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`), // Payload EventMessageID might be different
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testDiscordMessageID, // ADD discord_message_id to metadata
				},
			},
			want:    []*message.Message{{}}, // Assuming a trace message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				// The payload expected to be unmarshaled
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id", // Keep payload as is
					// DiscordMessageID is NOT expected in the payload for the handler
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						// Simulate unmarshalling the payload
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				// MODIFIED EXPECTATION: Expect testDiscordMessageID (string) from metadata
				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: true}, nil).
					Times(1)

				// Expect CreateResultMessage to be called with the original message (for metadata copy)
				// and a map payload (the trace payload)
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(), // Expect the original message 'msg' passed to the handler (for metadata)
						gomock.Any(), // Expect the trace payload (which is a map)
						roundevents.RoundTraceEvent,
					).
					Return(&message.Message{}, nil). // Return a dummy message
					Times(1)
			},
		},
		{
			name: "delete_round_event_embed_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testDiscordMessageID, // ADD discord_message_id to metadata
				},
			},
			want:    nil, // No message published on error
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				// MODIFIED EXPECTATION: Expect testDiscordMessageID (string) from metadata
				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{}, errors.New("failed to delete round event embed")).
					Times(1)

				// No CreateResultMessage expected in the error case
			},
		},
		{
			name: "delete_round_event_embed_returns_false",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testDiscordMessageID, // ADD discord_message_id to metadata
				},
			},
			want:    []*message.Message{{}}, // Still expect a trace message even if delete wasn't successful
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				// MODIFIED EXPECTATION: Expect testDiscordMessageID (string) from metadata
				// Simulate delete function returning success: false
				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: false, Error: errors.New("discord error")}, nil). // Include a dummy error in result
					Times(1)

				// Expect CreateResultMessage to be called with the original message and trace payload
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(), // Original message
						gomock.Any(), // Trace payload map
						roundevents.RoundTraceEvent,
					).
					Return(&message.Message{}, nil). // Return a dummy message
					Times(1)
			},
		},
		{
			name: "create_result_message_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testDiscordMessageID, // ADD discord_message_id to metadata
				},
			},
			want:    nil, // No message published on error
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					AnyTimes()

				// MODIFIED EXPECTATION: Expect testDiscordMessageID (string) from metadata
				mockDeleteRoundManager.EXPECT().
					DeleteRoundEventEmbed(gomock.Any(), testDiscordMessageID, gomock.Any()).
					Return(deleteround.DeleteRoundOperationResult{Success: true}, nil).
					Times(1)

				// Simulate CreateResultMessage failure
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(), // Original message
						gomock.Any(), // Trace payload map
						roundevents.RoundTraceEvent,
					).
					Return(nil, errors.New("failed to create result message")). // Return error
					Times(1)
			},
		},
		// Add test cases for missing/invalid metadata if desired
		{
			name: "missing_discord_message_id_in_metadata",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					// discord_message_id is missing
				},
			},
			want:    nil, // Expect error
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// No calls to RoundDiscord or DeleteRoundManager expected because metadata check fails first
				// No calls to CreateResultMessage expected
			},
		},
		{
			name: "empty_discord_message_id_in_metadata",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "event_message_id": "some_other_id"}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": "", // Empty value
				},
			},
			want:    nil, // Expect error
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers, mockDeleteRoundManager *mocks.MockDeleteRoundManager) {
				expectedPayload := roundevents.RoundDeletedPayload{
					RoundID:        testRoundID,
					EventMessageID: "some_other_id",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundDeletedPayload{})).
					DoAndReturn(func(msg *message.Message, v any) error {
						*v.(*roundevents.RoundDeletedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				// No calls to RoundDiscord or DeleteRoundManager expected because metadata check fails first
				// No calls to CreateResultMessage expected
			},
		},
		// Note: Test for non-string metadata value might be difficult for map[string]string metadata
		// unless you specifically put a non-string type via reflection or unsafe operations,
		// which isn't typical for Watermill.
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
				Logger: mockLogger,
				Config: &config.Config{ // Provide a non-nil config with a Discord channel ID
					Discord: config.DiscordConfig{
						EventChannelID: "dummy_channel_id", // Provide a dummy channel ID for the config
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

			got, err := h.HandleRoundDeleted(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare message content instead of pointers
			if !compareMessages(got, tt.want) {
				// Create better error message with actual content
				gotStr := make([]string, len(got))
				wantStr := make([]string, len(tt.want))

				for i, msg := range got {
					if msg != nil {
						gotStr[i] = string(msg.Payload)
					} else {
						gotStr[i] = "<nil>"
					}
				}

				for i, msg := range tt.want {
					if msg != nil {
						wantStr[i] = string(msg.Payload)
					} else {
						wantStr[i] = "<nil>"
					}
				}

				t.Errorf("HandleRoundDeleted() messages don't match.\nGot payloads: %v\nWant payloads: %v", gotStr, wantStr)
			}
		})
	}
}
