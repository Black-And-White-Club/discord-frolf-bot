package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundCreateRequested(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_round_create_request",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"guild_id": "123456789", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user123", "channel_id": "channel123"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.CreateRoundRequestedPayload{
					GuildID:     "123456789",
					Title:       "Test Round",
					Description: *roundtypes.DescriptionPtr("Test Description"),
					Location:    *roundtypes.LocationPtr("Test Location"),
					StartTime:   "2024-01-01T12:00:00Z",
					UserID:      "user123",
					ChannelID:   "channel123",
				}

				// Make sure this is called by the wrapper
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.CreateRoundRequestedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.CreateRoundRequestedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				// Change return to match the expected type - use an empty struct instead of nil
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(0) // Not called in successful case

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundCreateRequest).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "create_result_message_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"guild_id": "123456789", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user123", "channel_id": "channel123"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.CreateRoundRequestedPayload{})).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundCreateRequest).
					Return(nil, errors.New("failed to create result message")).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "create_result_message_fails_and_update_interaction_response_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"guild_id": "123456789", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user123", "channel_id": "channel123"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.CreateRoundRequestedPayload{})).
					Return(nil).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundCreateRequest).
					Return(nil, errors.New("failed to create result message")).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
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
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper)

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

			got, err := h.HandleRoundCreateRequested(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// For successful cases we only need to know a message was produced, not pointer equality
			if !tt.wantErr && tt.name == "successful_round_create_request" {
				if len(got) != 1 || got[0] == nil {
					t.Errorf("HandleRoundCreateRequested() expected one non-nil message, got %v", got)
				}
			} else if !reflect.DeepEqual(got, tt.want) {
				// Preserve existing behavior for other test cases
				t.Errorf("HandleRoundCreateRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	startTime := sharedtypes.StartTime(parsedTime)

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_round_creation",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id":             "correlation_id",
					"interaction_correlation_id": "correlation_id", // Function looks for this key
				},
			},
			want:    []*message.Message{{}}, // Assuming a message is returned
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// Mock UpdateInteractionResponse call
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)

				// Mock SendRoundEventEmbed call with proper return value
				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: "1344376922888474625",
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage, // Return actual discord message
					}, nil).
					Times(1)

				// Mock CreateResultMessage call
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundEventMessageIDUpdate).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "update_interaction_response_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id":             "correlation_id",
					"interaction_correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}}, // Function continues even if UpdateInteractionResponse fails
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// Mock UpdateInteractionResponse to fail
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
					Times(1)

				// Mock SendRoundEventEmbed call - function continues despite UpdateInteractionResponse failure
				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: "1344376922888474625",
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage,
					}, nil).
					Times(1)

				// Mock CreateResultMessage call
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundEventMessageIDUpdate).
					Return(&message.Message{}, nil).
					Times(1)
			},
		},
		{
			name: "send_round_event_embed_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id":             "correlation_id",
					"interaction_correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// Mock UpdateInteractionResponse call
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)

				// Mock SendRoundEventEmbed to fail
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to send round event embed")).
					Times(1)
			},
		},
		{
			name: "send_round_event_embed_returns_result_with_error",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id":             "correlation_id",
					"interaction_correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// Mock UpdateInteractionResponse call
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)

				// Mock SendRoundEventEmbed to return result with error
				result := createround.CreateRoundOperationResult{Error: errors.New("error in result")}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(result, nil).
					Times(1)
			},
		},
		{
			name: "create_result_message_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id":             "correlation_id",
					"interaction_correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// Mock UpdateInteractionResponse call
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)

				// Mock SendRoundEventEmbed to succeed
				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: "1344376922888474625",
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage,
					}, nil).
					Times(1)

				// Mock CreateResultMessage to fail
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundEventMessageIDUpdate).
					Return(nil, errors.New("failed to create result message")).
					Times(1)
			},
		},
		{
			name: "missing_interaction_correlation_id",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "title": "Test Round", "description": "Test Description", "location": "Test Location", "start_time": "2024-01-01T12:00:00Z", "user_id": "user_id"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					// Missing interaction_correlation_id
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     testRoundID,
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   &startTime,
						UserID:      sharedtypes.DiscordID("user_id"),
					},
					ChannelID: "1344376922888474625",
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				// No UpdateInteractionResponse call expected since interaction_correlation_id is missing

				// Mock SendRoundEventEmbed to succeed
				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: "1344376922888474625",
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage,
					}, nil).
					Times(1)

				// Mock CreateResultMessage call
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundEventMessageIDUpdate).
					Return(&message.Message{}, nil).
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
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper)

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

			got, err := h.HandleRoundCreated(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundCreated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_round_creation_failed",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"reason": "Test Reason"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundCreationFailedPayload{
					Reason: "Test Reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "update_interaction_response_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"reason": "Test Reason"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundCreationFailedPayload{
					Reason: "Test Reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
					Times(1)
			},
		},
		{
			name: "different_reason_for_failure",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"reason": "Another Test Reason"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundCreationFailedPayload{
					Reason: "Another Test Reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
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
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper)

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

			got, err := h.HandleRoundCreationFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundCreationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidationFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_round_validation_failed",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"error_message": ["Error 1", "Error 2"]}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundValidationFailedPayload{
					ErrorMessage: []string{"Error 1", "Error 2"},
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundValidationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundValidationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "update_interaction_response_fails",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"error_message": ["Error 1", "Error 2"]}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundValidationFailedPayload{
					ErrorMessage: []string{"Error 1", "Error 2"},
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundValidationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundValidationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
					Times(1)
			},
		},
		{
			name: "different_error_messages",
			msg: &message.Message{
				UUID:    "2",
				Payload: []byte(`{"error_message": ["Error A", "Error B"]}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id_2",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundValidationFailedPayload{
					ErrorMessage: []string{"Error A", "Error B"},
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundValidationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundValidationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id_2", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
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
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockRoundDiscord, mockHelper)

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

			got, err := h.HandleRoundValidationFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundValidationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
