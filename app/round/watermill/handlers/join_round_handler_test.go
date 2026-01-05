package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
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

func TestRoundHandlers_HandleRoundParticipantJoinRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("123456789")

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_accepted_response",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "accepted",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Verify the correct backend payload is constructed
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						backendPayload, ok := payload.(roundevents.ParticipantJoinRequestPayloadV1)
						if !ok {
							t.Errorf("Expected roundevents.ParticipantJoinRequestPayloadV1, got %T", payload)
						}
						if backendPayload.RoundID != testRoundID {
							t.Errorf("Expected RoundID %v, got %v", testRoundID, backendPayload.RoundID)
						}
						if backendPayload.UserID != sharedtypes.DiscordID(testUserID) {
							t.Errorf("Expected UserID %v, got %v", testUserID, backendPayload.UserID)
						}
						if backendPayload.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected Response %v, got %v", roundtypes.ResponseAccept, backendPayload.Response)
						}
						if *backendPayload.TagNumber != 0 {
							t.Errorf("Expected TagNumber 0, got %v", *backendPayload.TagNumber)
						}
						if *backendPayload.JoinedLate {
							t.Errorf("Expected JoinedLate false, got %v", *backendPayload.JoinedLate)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "successful_declined_response",
			msg: &message.Message{
				UUID:    "2",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "declined",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Verify correct backend payload for declined response
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						backendPayload, ok := payload.(roundevents.ParticipantJoinRequestPayloadV1)
						if !ok {
							t.Errorf("Expected roundevents.ParticipantJoinRequestPayloadV1, got %T", payload)
						}
						if backendPayload.Response != roundtypes.ResponseDecline {
							t.Errorf("Expected Response %v, got %v", roundtypes.ResponseDecline, backendPayload.Response)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "successful_tentative_response",
			msg: &message.Message{
				UUID:    "3",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "tentative",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Verify correct backend payload for tentative response
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						backendPayload, ok := payload.(roundevents.ParticipantJoinRequestPayloadV1)
						if !ok {
							t.Errorf("Expected roundevents.ParticipantJoinRequestPayloadV1, got %T", payload)
						}
						if backendPayload.Response != roundtypes.ResponseTentative {
							t.Errorf("Expected Response %v, got %v", roundtypes.ResponseTentative, backendPayload.Response)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "invalid_response_defaults_to_accept",
			msg: &message.Message{
				UUID:    "4",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "invalid",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Verify invalid response defaults to accepted
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						backendPayload, ok := payload.(roundevents.ParticipantJoinRequestPayloadV1)
						if !ok {
							t.Errorf("Expected roundevents.ParticipantJoinRequestPayloadV1, got %T", payload)
						}
						if backendPayload.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected Response %v, got %v", roundtypes.ResponseAccept, backendPayload.Response)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "with_joined_late_flag",
			msg: &message.Message{
				UUID:    "5",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `", "joined_late": true}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "accepted",
				},
			},
			want:    []*message.Message{{}},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				joinedLate := true
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID:    testRoundID,
					UserID:     testUserID,
					JoinedLate: &joinedLate,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Verify JoinedLate flag is passed correctly
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
					).
					DoAndReturn(func(_ *message.Message, payload any, _ string) (*message.Message, error) {
						backendPayload, ok := payload.(roundevents.ParticipantJoinRequestPayloadV1)
						if !ok {
							t.Errorf("Expected roundevents.ParticipantJoinRequestPayloadV1, got %T", payload)
						}
						if !*backendPayload.JoinedLate {
							t.Errorf("Expected JoinedLate true, got %v", *backendPayload.JoinedLate)
						}
						return &message.Message{}, nil
					}).
					Times(1)
			},
		},
		{
			name: "create_result_message_error",
			msg: &message.Message{
				UUID:    "6",
				Payload: []byte(`{"round_id": "` + testRoundID.String() + `", "user_id": "` + string(testUserID) + `"}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					"response":       "accepted",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
					RoundID: testRoundID,
					UserID:  testUserID,
				}

				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) = expectedPayload
						return nil
					}).
					Times(1)

				// Simulate error when creating result message
				mockHelper.EXPECT().
					CreateResultMessage(
						gomock.Any(),
						gomock.Any(),
						roundevents.RoundParticipantJoinRequestedV1,
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

			got, err := h.HandleRoundParticipantJoinRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundParticipantJoinRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundParticipantJoined(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testMessageID := "12345"
	testChannelID := "test-channel-id"

	tag1 := sharedtypes.TagNumber(1)

	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *util_mocks.MockHelpers, *mocks.MockRoundDiscordInterface, *mocks.MockRoundRsvpManager, *config.Config)
	}{
		{
			name: "successful_participant_joined",
			msg: &message.Message{
				UUID: "1",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"event_message_id": "` + testMessageID + `",
					"accepted_participants": [{"user_id": "user1", "tag_number": 1}],
					"declined_participants": [{"user_id": "user2"}],
					"tentative_participants": [{"user_id": "user3"}]
			}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testMessageID, // Add the required metadata
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockRoundRsvpManager *mocks.MockRoundRsvpManager, mockConfig *config.Config) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayloadV1) = roundevents.ParticipantJoinedPayloadV1{
							RoundID:        testRoundID,
							EventMessageID: testMessageID,
							AcceptedParticipants: []roundtypes.Participant{
								{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
							},
							DeclinedParticipants: []roundtypes.Participant{
								{UserID: "user2", Response: roundtypes.ResponseDecline},
							},
							TentativeParticipants: []roundtypes.Participant{
								{UserID: "user3", Response: roundtypes.ResponseTentative},
							},
						}
						return nil
					}).
					Times(1)

				mockConfig.Discord.EventChannelID = testChannelID
				mockRoundDiscord.EXPECT().GetRoundRsvpManager().Return(mockRoundRsvpManager).Times(1)
				mockRoundRsvpManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testMessageID, // This should now match the metadata value
					[]roundtypes.Participant{
						{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
					},
					[]roundtypes.Participant{
						{UserID: "user2", Response: roundtypes.ResponseDecline},
					},
					[]roundtypes.Participant{
						{UserID: "user3", Response: roundtypes.ResponseTentative},
					},
				).Return(roundrsvp.RoundRsvpOperationResult{}, nil).Times(1)
			},
		},
		{
			name: "successful_late_join",
			msg: &message.Message{
				UUID: "2",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"event_message_id": "` + testMessageID + `",
					"accepted_participants": [{"user_id": "user1", "tag_number": 1}],
					"declined_participants": [{"user_id": "user2"}],
					"tentative_participants": [{"user_id": "user3"}],
					"joined_late": true
			}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testMessageID, // Add the required metadata
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockRoundRsvpManager *mocks.MockRoundRsvpManager, mockConfig *config.Config) {
				joinedLate := true
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayloadV1) = roundevents.ParticipantJoinedPayloadV1{
							RoundID:        testRoundID,
							EventMessageID: testMessageID,
							AcceptedParticipants: []roundtypes.Participant{
								{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
							},
							DeclinedParticipants: []roundtypes.Participant{
								{UserID: "user2", Response: roundtypes.ResponseDecline},
							},
							TentativeParticipants: []roundtypes.Participant{
								{UserID: "user3", Response: roundtypes.ResponseTentative},
							},
							JoinedLate: &joinedLate,
						}
						return nil
					}).
					Times(1)

				mockConfig.Discord.EventChannelID = testChannelID
				mockScoreRoundManager := mocks.NewMockScoreRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetScoreRoundManager().Return(mockScoreRoundManager).Times(1)
				mockScoreRoundManager.EXPECT().AddLateParticipantToScorecard(
					gomock.Any(),
					testChannelID,
					testMessageID,
					[]roundtypes.Participant{
						{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
					},
				).Return(scoreround.ScoreRoundOperationResult{}, nil).Times(1)
			},
		},
		{
			name: "update_embed_error",
			msg: &message.Message{
				UUID: "3",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"event_message_id": "` + testMessageID + `",
					"accepted_participants": [{"user_id": "user1", "tag_number": 1}],
					"declined_participants": [{"user_id": "user2"}],
					"tentative_participants": [{"user_id": "user3"}]
			}`),
				Metadata: message.Metadata{
					"correlation_id":     "correlation_id",
					"discord_message_id": testMessageID, // Add the required metadata
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockRoundRsvpManager *mocks.MockRoundRsvpManager, mockConfig *config.Config) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayloadV1) = roundevents.ParticipantJoinedPayloadV1{
							RoundID:        testRoundID,
							EventMessageID: testMessageID,
							AcceptedParticipants: []roundtypes.Participant{
								{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
							},
							DeclinedParticipants: []roundtypes.Participant{
								{UserID: "user2", Response: roundtypes.ResponseDecline},
							},
							TentativeParticipants: []roundtypes.Participant{
								{UserID: "user3", Response: roundtypes.ResponseTentative},
							},
						}
						return nil
					}).
					Times(1)

				mockConfig.Discord.EventChannelID = testChannelID
				mockRoundDiscord.EXPECT().GetRoundRsvpManager().Return(mockRoundRsvpManager).Times(1)
				mockRoundRsvpManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					testMessageID, // This should now match the metadata value
					[]roundtypes.Participant{
						{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
					},
					[]roundtypes.Participant{
						{UserID: "user2", Response: roundtypes.ResponseDecline},
					},
					[]roundtypes.Participant{
						{UserID: "user3", Response: roundtypes.ResponseTentative},
					},
				).Return(roundrsvp.RoundRsvpOperationResult{}, errors.New("failed to update embed")).Times(1)
			},
		},
		{
			name: "missing_discord_message_id",
			msg: &message.Message{
				UUID: "4",
				Payload: []byte(`{
					"round_id": "` + testRoundID.String() + `",
					"event_message_id": "` + testMessageID + `",
					"accepted_participants": [{"user_id": "user1", "tag_number": 1}],
					"declined_participants": [{"user_id": "user2"}],
					"tentative_participants": [{"user_id": "user3"}]
			}`),
				Metadata: message.Metadata{
					"correlation_id": "correlation_id",
					// Missing discord_message_id
				},
			},
			want:    nil,
			wantErr: false, // Function continues with empty messageID
			setup: func(ctrl *gomock.Controller, mockHelper *util_mocks.MockHelpers, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockRoundRsvpManager *mocks.MockRoundRsvpManager, mockConfig *config.Config) {
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayloadV1{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayloadV1) = roundevents.ParticipantJoinedPayloadV1{
							RoundID:        testRoundID,
							EventMessageID: testMessageID,
							AcceptedParticipants: []roundtypes.Participant{
								{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
							},
							DeclinedParticipants: []roundtypes.Participant{
								{UserID: "user2", Response: roundtypes.ResponseDecline},
							},
							TentativeParticipants: []roundtypes.Participant{
								{UserID: "user3", Response: roundtypes.ResponseTentative},
							},
						}
						return nil
					}).
					Times(1)

				mockConfig.Discord.EventChannelID = testChannelID
				mockRoundDiscord.EXPECT().GetRoundRsvpManager().Return(mockRoundRsvpManager).Times(1)
				mockRoundRsvpManager.EXPECT().UpdateRoundEventEmbed(
					gomock.Any(),
					testChannelID,
					"", // Empty messageID when metadata is missing
					[]roundtypes.Participant{
						{UserID: "user1", TagNumber: &tag1, Response: roundtypes.ResponseAccept},
					},
					[]roundtypes.Participant{
						{UserID: "user2", Response: roundtypes.ResponseDecline},
					},
					[]roundtypes.Participant{
						{UserID: "user3", Response: roundtypes.ResponseTentative},
					},
				).Return(roundrsvp.RoundRsvpOperationResult{}, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockRoundRsvpManager := mocks.NewMockRoundRsvpManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")
			mockConfig := &config.Config{}

			// Setup test-specific expectations
			tt.setup(ctrl, mockHelper, mockRoundDiscord, mockRoundRsvpManager, mockConfig)

			h := &RoundHandlers{
				Logger:       mockLogger,
				Config:       mockConfig,
				Helpers:      mockHelper,
				RoundDiscord: mockRoundDiscord,
				Tracer:       mockTracer,
				Metrics:      mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleRoundParticipantJoined(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundParticipantJoined() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundParticipantJoined() = %v, want %v", got, tt.want)
			}
		})
	}
}
