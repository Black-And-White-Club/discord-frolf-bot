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
	updateround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/update_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundUpdateRequested(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round Update")
	testDescription := roundtypes.Description("Updated Description")

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_update_request",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"title": "` + string(testTitle) + `",
					"description": "` + string(testDescription) + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.DiscordRoundUpdateRequestPayload{
					RoundID:     testRoundID,
					Title:       &testTitle,
					Description: &testDescription,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundUpdateRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundUpdateRequestPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundUpdateRequest,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						updatePayload, ok := payload.(*discordroundevents.DiscordRoundUpdateRequestPayload)
						if !ok {
							t.Errorf("Expected *discordroundevents.DiscordRoundUpdateRequestPayload, got %T", payload)
						}
						if updatePayload.RoundID != testRoundID {
							t.Errorf("Expected RoundID %v, got %v", testRoundID, updatePayload.RoundID)
						}
						if updatePayload.Title != &testTitle {
							t.Errorf("Expected Title %v, got %v", testTitle, updatePayload.Title)
						}
						if *updatePayload.Description != testDescription {
							t.Errorf("Expected Description %v, got %v", testDescription, *updatePayload.Description)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "create_result_message_error",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"title": "` + string(testTitle) + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.DiscordRoundUpdateRequestPayload{
					RoundID: testRoundID,
					Title:   &testTitle,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundUpdateRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundUpdateRequestPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundUpdateRequest,
					).
					Return(nil, errors.New("failed to create message")).
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

			// Setup test-specific expectations
			tt.setup(ctrl, mockHelper)

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

			got, err := h.HandleRoundUpdateRequested(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdateRequested() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testChannelID := "test-channel-id"
	testTitle := roundtypes.Title("Updated Round Title")
	testDescription := roundtypes.Description("Updated Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.LocationPtr("Updated Location")

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockRoundDiscordInterface, *mocks.MockUpdateRoundManager)
	}{
		{
			name: "successful_round_updated",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"channel_id": "` + testChannelID + `",
					"title": "` + string(testTitle) + `",
					"description": "` + string(testDescription) + `",
					"start_time": "2023-05-01T14:00:00Z",
					"location": "` + string(*testLocation) + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				expectedPayload := discordroundevents.DiscordRoundUpdatedPayload{
					RoundID:     testRoundID,
					ChannelID:   testChannelID,
					Title:       &testTitle,
					Description: &testDescription,
					StartTime:   &testStartTime,
					Location:    testLocation,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				// Return a valid UpdateRoundOperationResult instead of struct{}
				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testRoundID,
					&testTitle,
					&testDescription,
					&testStartTime,
					testLocation,
				).Return(updateround.UpdateRoundOperationResult{}, nil).Times(1) // Correct return type
			},
		},
		{
			name: "update_with_nil_fields",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"channel_id": "` + testChannelID + `",
					"title": "` + string(testTitle) + `",
					"location": "` + string(*testLocation) + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				expectedPayload := discordroundevents.DiscordRoundUpdatedPayload{
					RoundID:   testRoundID,
					ChannelID: testChannelID,
					Title:     &testTitle,
					Location:  testLocation,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				// Return a valid UpdateRoundOperationResult instead of struct{}
				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testRoundID,
					&testTitle,
					nil,
					nil,
					testLocation,
				).Return(updateround.UpdateRoundOperationResult{}, nil).Times(1) // Correct return type
			},
		},
		{
			name: "update_embed_error",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
            "round_id": "` + testRoundID.String() + `",
            "channel_id": "` + testChannelID + `",
            "title": "` + string(testTitle) + `"
        }`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockUpdateRoundManager *mocks.MockUpdateRoundManager) {
				expectedPayload := discordroundevents.DiscordRoundUpdatedPayload{
					RoundID:   testRoundID,
					ChannelID: testChannelID,
					Title:     &testTitle,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundUpdatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundUpdatedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().GetUpdateRoundManager().Return(mockUpdateRoundManager).Times(1)

				mockUpdateRoundManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testRoundID,
					&testTitle, // Correct pointer type
					nil,
					nil,
					nil, // Correct nil instead of ""
				).Return(updateround.UpdateRoundOperationResult{
					Error: errors.New("failed to update embed"),
				}, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockUpdateRoundManager := mocks.NewMockUpdateRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockHelper, mockRoundDiscord, mockUpdateRoundManager)

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

			got, err := h.HandleRoundUpdated(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateFailed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testError := "Round update failed due to database error"
	testTitle := roundtypes.Title("Test Title")
	testDescription := roundtypes.Description("Test Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.Location("Test Location")
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "handle_update_failed_with_full_payload",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"title": "` + string(testTitle) + `",
					"description": "` + string(testDescription) + `",
					"start_time": "2023-05-01T14:00:00Z",
					"location": "` + string(testLocation) + `",
					"user_id": "` + string(testUserID) + `",
					"error": "` + testError + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: &testDescription,
							StartTime:   &testStartTime,
							Location:    &testLocation,
							UserID:      testUserID,
						},
					},
					Error: testError,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundUpdateErrorPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundUpdateErrorPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "handle_update_failed_minimal_payload",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"error": "` + testError + `"
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
						},
					},
					Error: testError,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundUpdateErrorPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundUpdateErrorPayload) = expectedPayload
						return nil
					}).
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

			tt.setup(ctrl, mockHelper)

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

			got, err := h.HandleRoundUpdateFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdateFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundUpdateValidationFailed(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testTitle := roundtypes.Title("Test Round")
	testDescription := roundtypes.Description("Test Description")
	parsedStartTime, _ := time.Parse(time.RFC3339, "2023-05-01T14:00:00Z")
	testStartTime := sharedtypes.StartTime(parsedStartTime)
	testLocation := roundtypes.Location("Test Location")
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "handle_validation_failed_full_payload",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"round_update_request_payload": {
						"round_id": "` + testRoundID.String() + `",
						"title": "` + string(testTitle) + `",
						"description": "` + string(testDescription) + `",
						"start_time": "2023-05-01T14:00:00Z",
						"location": "` + string(testLocation) + `",
						"user_id": "` + string(testUserID) + `"
					},
					"validation_errors": ["Title is required", "Start time is invalid"]
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID:     testRoundID,
							Title:       testTitle,
							Description: &testDescription,
							StartTime:   &testStartTime,
							Location:    &testLocation,
							UserID:      testUserID,
						},
					},
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundUpdateValidatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundUpdateValidatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "handle_validation_failed_minimal_payload",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
					"round_update_request_payload": {
						"round_id": "` + testRoundID.String() + `"
					},
					"validation_errors": ["Missing required fields"]
				}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
						},
					},
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundUpdateValidatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundUpdateValidatedPayload) = expectedPayload
						return nil
					}).
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

			tt.setup(ctrl, mockHelper)

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

			got, err := h.HandleRoundUpdateValidationFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundUpdateValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundUpdateValidationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
