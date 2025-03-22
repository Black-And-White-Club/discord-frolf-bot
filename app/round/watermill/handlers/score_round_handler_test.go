package roundhandlers

import (
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockScoreRoundManager := mocks.NewMockScoreRoundManager(ctrl)

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
			name: "successful score update",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"participant": "user1",
					"score": 72,
					"channel_id": "channel123"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"participant_score_updated","status":"confirmation_sent","participant":"user1","score":72}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				score := 72
				payload := &roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     123,
					Participant: "user1",
					Score:       score,
					ChannelID:   "channel123",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						scorePayload := p.(*roundevents.ParticipantScoreUpdatedPayload)
						scorePayload.RoundID = payload.RoundID
						scorePayload.Participant = payload.Participant
						scorePayload.Score = payload.Score
						scorePayload.ChannelID = payload.ChannelID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateConfirmation("channel123", roundtypes.UserID("user1"), gomock.Any()).
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.RoundID,
					"event_type":  "participant_score_updated",
					"status":      "confirmation_sent",
					"participant": payload.Participant,
					"score":       payload.Score,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"participant_score_updated","status":"confirmation_sent","participant":"user1","score":72}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(traceMsg, nil).
					Times(1)
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
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to unmarshal payload")).
					Times(1)
			},
		},
		{
			name: "failed to send score update confirmation",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"participant": "user1",
					"score": 72,
					"channel_id": "channel123"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"participant_score_updated","status":"confirmation_sent","participant":"user1","score":72}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				score := 72
				payload := &roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     123,
					Participant: "user1",
					Score:       score,
					ChannelID:   "channel123",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						scorePayload := p.(*roundevents.ParticipantScoreUpdatedPayload)
						scorePayload.RoundID = payload.RoundID
						scorePayload.Participant = payload.Participant
						scorePayload.Score = payload.Score
						scorePayload.ChannelID = payload.ChannelID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateConfirmation("channel123", roundtypes.UserID("user1"), gomock.Any()).
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.RoundID,
					"event_type":  "participant_score_updated",
					"status":      "confirmation_sent",
					"participant": payload.Participant,
					"score":       payload.Score,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"participant_score_updated","status":"confirmation_sent","participant":"user1","score":72}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(traceMsg, nil).
					Times(1)
			},
		},
		{
			name: "failed to create trace event message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"participant": "user1",
					"score": 72,
					"channel_id": "channel123"
				}`)),
			},
			want:    []*message.Message{},
			wantErr: false,
			setup: func() {
				score := 72
				payload := &roundevents.ParticipantScoreUpdatedPayload{
					RoundID:     123,
					Participant: "user1",
					Score:       score,
					ChannelID:   "channel123",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						scorePayload := p.(*roundevents.ParticipantScoreUpdatedPayload)
						scorePayload.RoundID = payload.RoundID
						scorePayload.Participant = payload.Participant
						scorePayload.Score = payload.Score
						scorePayload.ChannelID = payload.ChannelID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateConfirmation("channel123", roundtypes.UserID("user1"), &score).
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.RoundID,
					"event_type":  "participant_score_updated",
					"status":      "confirmation_sent",
					"participant": payload.Participant,
					"score":       payload.Score,
				}

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(nil, errors.New("failed to create trace message")).
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
			}
			got, err := h.HandleParticipantScoreUpdated(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleParticipantScoreUpdated() = %v, want nil", got)
				}
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleParticipantScoreUpdated() returned %d messages, want %d", len(got), len(tt.want))
			} else if len(got) > 0 && len(tt.want) > 0 {
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

func TestRoundHandlers_HandleScoreUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockScoreRoundManager := mocks.NewMockScoreRoundManager(ctrl)

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
			name: "successful error handling",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"error": "Invalid score format",
					"score_update_request": {
						"round_id": 123,
						"participant": "user1"
					}
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"score_update_error","status":"error_notification_sent","participant":"user1","error":"Invalid score format"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundScoreUpdateErrorPayload{
					Error: "Invalid score format",
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     123,
						Participant: "user1",
					},
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						errorPayload := p.(*roundevents.RoundScoreUpdateErrorPayload)
						errorPayload.Error = payload.Error
						errorPayload.ScoreUpdateRequest = payload.ScoreUpdateRequest
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateError(roundtypes.UserID("user1"), "Invalid score format").
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.ScoreUpdateRequest.RoundID,
					"event_type":  "score_update_error",
					"status":      "error_notification_sent",
					"participant": payload.ScoreUpdateRequest.Participant,
					"error":       payload.Error,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"score_update_error","status":"error_notification_sent","participant":"user1","error":"Invalid score format"}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(traceMsg, nil).
					Times(1)
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
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(errors.New("failed to unmarshal payload")).
					Times(1)
			},
		},
		{
			name: "empty error message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"error": "",
					"score_update_request": {
						"round_id": 123,
						"participant": "user1"
					}
				}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundScoreUpdateErrorPayload{
					Error: "",
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     123,
						Participant: "user1",
					},
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						errorPayload := p.(*roundevents.RoundScoreUpdateErrorPayload)
						errorPayload.Error = payload.Error
						errorPayload.ScoreUpdateRequest = payload.ScoreUpdateRequest
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "failed to send error notification",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"error": "Invalid score format",
					"score_update_request": {
						"round_id": 123,
						"participant": "user1"
					}
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"score_update_error","status":"error_notification_sent","participant":"user1","error":"Invalid score format"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundScoreUpdateErrorPayload{
					Error: "Invalid score format",
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     123,
						Participant: "user1",
					},
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						errorPayload := p.(*roundevents.RoundScoreUpdateErrorPayload)
						errorPayload.Error = payload.Error
						errorPayload.ScoreUpdateRequest = payload.ScoreUpdateRequest
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateError(roundtypes.UserID("user1"), "Invalid score format").
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.ScoreUpdateRequest.RoundID,
					"event_type":  "score_update_error",
					"status":      "error_notification_sent",
					"participant": payload.ScoreUpdateRequest.Participant,
					"error":       payload.Error,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"score_update_error","status":"error_notification_sent","participant":"user1","error":"Invalid score format"}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(traceMsg, nil).
					Times(1)
			},
		},
		{
			name: "failed to create trace event message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"error": "Invalid score format",
					"score_update_request": {
						"round_id": 123,
						"participant": "user1"
					}
				}`)),
			},
			want:    []*message.Message{},
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundScoreUpdateErrorPayload{
					Error: "Invalid score format",
					ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayload{
						RoundID:     123,
						Participant: "user1",
					},
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						errorPayload := p.(*roundevents.RoundScoreUpdateErrorPayload)
						errorPayload.Error = payload.Error
						errorPayload.ScoreUpdateRequest = payload.ScoreUpdateRequest
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateError(roundtypes.UserID("user1"), "Invalid score format").
					Return(nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":    payload.ScoreUpdateRequest.RoundID,
					"event_type":  "score_update_error",
					"status":      "error_notification_sent",
					"participant": payload.ScoreUpdateRequest.Participant,
					"error":       payload.Error,
				}

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Eq(tracePayload), roundevents.RoundTraceEvent).
					Return(nil, errors.New("failed to create trace message")).
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
			}
			got, err := h.HandleScoreUpdateError(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleScoreUpdateError() = %v, want nil", got)
				}
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleScoreUpdateError() returned %d messages, want %d", len(got), len(tt.want))
			} else if len(got) > 0 && len(tt.want) > 0 {
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
