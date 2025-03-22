package roundhandlers

import (
	"errors"
	"reflect"
	"testing"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

// Helper function to convert int to *int
func intPtr(i int) *int {
	return &i
}

func TestRoundHandlers_HandleRoundParticipantJoinRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)

	type fields struct {
		Logger       observability.Logger
		Helpers      *utils.MockHelpers
		RoundDiscord *mocks.MockRoundDiscordInterface
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "successful normal join",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("backend-1", []byte(`{"round_id": 123, "user_id": "user456"}`)),
			},
			want: func() []*message.Message {
				return []*message.Message{message.NewMessage("backend-1", nil)}
			}(),
			wantErr: false,
			setup: func() {
				expectedPayload := discordroundevents.DiscordRoundParticipantJoinRequestPayload{
					RoundID: 123,
					UserID:  "user456",
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundParticipantJoinRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload) = expectedPayload
						return nil
					}).Times(1)

				tagNumber := 0
				falseVar := false
				expectedBackendPayload := roundevents.ParticipantJoinRequestPayload{
					RoundID:    123,
					UserID:     "user456",
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tagNumber,
					JoinedLate: &falseVar,
				}

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedBackendPayload), roundevents.RoundParticipantJoinRequest).
					Return(message.NewMessage("backend-1", nil), nil).Times(1)
			},
		},
		{
			name: "successful participant join request - explicit response",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: func() *message.Message {
					msg := message.NewMessage("1", []byte(`{"round_id": 123, "user_id": "user456"}`))
					msg.Metadata.Set("response", "tentative")
					return msg
				}(),
			},
			want: func() []*message.Message {
				return []*message.Message{message.NewMessage(roundevents.RoundParticipantJoinRequest, []byte(`{"round_id":123,"user_id":"user456","response":"tentative","tag_number":0}`))}
			}(),
			wantErr: false,
			setup: func() {
				expectedPayload := discordroundevents.DiscordRoundParticipantJoinRequestPayload{
					RoundID: 123,
					UserID:  "user456",
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundParticipantJoinRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload) = expectedPayload
						return nil
					}).Times(1)

				msg := message.NewMessage(roundevents.RoundParticipantJoinRequest, []byte(`{"round_id":123,"user_id":"user456","response":"tentative","tag_number":0}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundParticipantJoinRequest).
					Return(msg, nil).Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to create result message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"round_id": 123, "user_id": "user456"}`)),
			},
			wantErr: true,
			setup: func() {
				expectedPayload := discordroundevents.DiscordRoundParticipantJoinRequestPayload{
					RoundID: 123,
					UserID:  "user456",
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundParticipantJoinRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload) = expectedPayload
						return nil
					}).Times(1)

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundParticipantJoinRequest).
					Return(nil, errors.New("failed to create result message")).
					Times(1)
			},
		},
		{
			name: "successful late join",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"round_id": 123, "user_id": "user456", "joined_late": true}`)),
			},
			want: func() []*message.Message {
				return []*message.Message{message.NewMessage("backend-1", nil)}
			}(),
			wantErr: false,
			setup: func() {
				expectedPayload := discordroundevents.DiscordRoundParticipantJoinRequestPayload{
					RoundID:    123,
					UserID:     "user456",
					JoinedLate: func() *bool { b := true; return &b }(),
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.DiscordRoundParticipantJoinRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.DiscordRoundParticipantJoinRequestPayload) = expectedPayload
						return nil
					}).Times(1)

				tagNumber := 0
				joinedLate := true
				expectedBackendPayload := roundevents.ParticipantJoinRequestPayload{
					RoundID:    123,
					UserID:     "user456",
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tagNumber,
					JoinedLate: &joinedLate,
				}

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(expectedBackendPayload), roundevents.RoundParticipantJoinRequest).
					Return(message.NewMessage("backend-1", nil), nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &RoundHandlers{
				Logger:       tt.fields.Logger,
				Helpers:      tt.fields.Helpers,
				RoundDiscord: tt.fields.RoundDiscord,
			}
			got, err := h.HandleRoundParticipantJoinRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleRoundParticipantJoinRequest() = %v, want nil", got)
				}
			} else if got == nil {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoinRequest() = nil, want %v", tt.want)
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoinRequest() returned %d messages, want %d", len(got), len(tt.want))
			} else {
				for i, wantMsg := range tt.want {
					if i >= len(got) {
						t.Errorf("Missing expected message at index %d", i)
						continue
					}

					gotMsg := got[i]
					if wantMsg.UUID != gotMsg.UUID {
						t.Errorf("Message UUID mismatch at index %d: got %s, want %s", i, gotMsg.UUID, wantMsg.UUID)
					}

					if string(wantMsg.Payload) != string(gotMsg.Payload) {
						t.Errorf("Message payload mismatch at index %d: got %s, want %s", i, string(gotMsg.Payload), string(wantMsg.Payload))
					}
				}
			}
		})
	}
}
func TestRoundHandlers_HandleRoundParticipantJoined(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockRoundRsvpManager := mocks.NewMockRoundRsvpManager(ctrl)

	// Create a test config with the required channel ID
	testConfig := &config.Config{
		Discord: config.DiscordConfig{
			ChannelID: "channel123",
		},
	}

	type fields struct {
		Logger       observability.Logger
		Helpers      *utils.MockHelpers
		RoundDiscord *mocks.MockRoundDiscordInterface
		Config       *config.Config
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "successful participant joined (normal join)",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
				Config:       testConfig,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"accepted_participants": [{"user_id": "user1"}],
					"event_message_id": "message456"
				}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := roundevents.ParticipantJoinedPayload{
					RoundID:              123,
					AcceptedParticipants: []roundtypes.Participant{{UserID: "user1"}},
					JoinedLate:           nil,
					EventMessageID:       "message456",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayload) = expectedPayload
						return nil
					}).Times(1)

				mockRoundDiscord.EXPECT().
					GetRoundRsvpManager().
					Return(mockRoundRsvpManager).
					Times(1)

				mockRoundRsvpManager.EXPECT().
					UpdateRoundEventEmbed("channel123", roundtypes.EventMessageID("message456"),
						expectedPayload.AcceptedParticipants, nil, nil).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "successful participant joined (late join)",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
				Config:       testConfig,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"accepted_participants": [{"user_id": "user1"}],
					"joined_late": true,
					"event_message_id": "message456"
				}`)),
			},
			wantErr: false,
			setup: func() {
				joinedLate := true
				expectedPayload := roundevents.ParticipantJoinedPayload{
					RoundID:              123,
					AcceptedParticipants: []roundtypes.Participant{{UserID: "user1"}},
					JoinedLate:           &joinedLate,
					EventMessageID:       "message456",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayload) = expectedPayload
						return nil
					}).Times(1)

				mockRoundDiscord.EXPECT().
					GetRoundRsvpManager().
					Return(mockRoundRsvpManager).
					Times(1)

				mockRoundRsvpManager.EXPECT().
					UpdateRoundEventEmbed("channel123", roundtypes.EventMessageID("message456"),
						expectedPayload.AcceptedParticipants, nil, nil).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
				Config:       testConfig,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to update round event embed",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
				Config:       testConfig,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"accepted_participants": [{"user_id": "user1", "tag_number": 1}],
					"declined_participants": [],
					"tentative_participants": [],
					"event_message_id": "message456"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				acceptedParticipants := []roundtypes.Participant{
					{UserID: "user1", TagNumber: intPtr(1)},
				}
				var declinedParticipants []roundtypes.Participant
				var tentativeParticipants []roundtypes.Participant

				expectedPayload := roundevents.ParticipantJoinedPayload{
					RoundID:               123,
					AcceptedParticipants:  acceptedParticipants,
					DeclinedParticipants:  declinedParticipants,
					TentativeParticipants: tentativeParticipants,
					EventMessageID:        "message456",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.ParticipantJoinedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.ParticipantJoinedPayload) = expectedPayload
						return nil
					}).Times(1)

				mockRoundDiscord.EXPECT().
					GetRoundRsvpManager().
					Return(mockRoundRsvpManager).
					Times(1)

				mockRoundRsvpManager.EXPECT().
					UpdateRoundEventEmbed("channel123", roundtypes.EventMessageID("message456"), acceptedParticipants, declinedParticipants, tentativeParticipants).
					Return(errors.New("update error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &RoundHandlers{
				Logger:       tt.fields.Logger,
				Helpers:      tt.fields.Helpers,
				RoundDiscord: tt.fields.RoundDiscord,
				Config:       tt.fields.Config,
			}
			got, err := h.HandleRoundParticipantJoined(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoined() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoined() = %v, want %v", got, tt.want)
			}
		})
	}
}
