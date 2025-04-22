package roundhandlers

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockDeleteRoundManager := mocks.NewMockDeleteRoundManager(ctrl)

	// Create a proper config with the required fields
	mockConfigObj := &config.Config{
		Discord: config.DiscordConfig{
			ChannelID: "channel123",
		},
	}

	type fields struct {
		Logger       *slog.Logger
		Config       *config.Config
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
			name: "successful round deleted",
			fields: fields{
				Logger:       mockLogger,
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"event_message_id": "message789"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_deleted","status":"embed_deleted","message_id":"message789"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        123,
					EventMessageID: "message789",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					Times(1)

				mockDeleteRoundManager.EXPECT().
					DeleteEmbed(gomock.Any(), roundtypes.EventMessageID("message789"), "channel123").
					Return(true, nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":   payload.RoundID,
					"event_type": "round_deleted",
					"status":     "embed_deleted",
					"message_id": payload.EventMessageID,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_deleted","status":"embed_deleted","message_id":"message789"}`))
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
				Config:       mockConfigObj,
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
			name: "missing round ID",
			fields: fields{
				Logger:       mockLogger,
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"event_message_id": "message789"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        0,
					EventMessageID: "message789",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "missing event message ID",
			fields: fields{
				Logger:       mockLogger,
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        123,
					EventMessageID: "",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "failed to delete embed message",
			fields: fields{
				Logger:       mockLogger,
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"event_message_id": "message789"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        123,
					EventMessageID: "message789",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					Times(1)

				mockDeleteRoundManager.EXPECT().
					DeleteEmbed(gomock.Any(), roundtypes.EventMessageID("message789"), "channel123").
					Return(false, errors.New("failed to delete embed")).
					Times(1)
			},
		},
		{
			name: "deletion not successful but no error",
			fields: fields{
				Logger:       mockLogger,
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"event_message_id": "message789"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_deleted","status":"embed_deleted","message_id":"message789"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        123,
					EventMessageID: "message789",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					Times(1)

				mockDeleteRoundManager.EXPECT().
					DeleteEmbed(gomock.Any(), roundtypes.EventMessageID("message789"), "channel123").
					Return(false, nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":   payload.RoundID,
					"event_type": "round_deleted",
					"status":     "embed_deleted",
					"message_id": payload.EventMessageID,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_deleted","status":"embed_deleted","message_id":"message789"}`))
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
				Config:       mockConfigObj,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"event_message_id": "message789"
				}`)),
			},
			want:    []*message.Message{},
			wantErr: false,
			setup: func() {
				payload := &roundevents.RoundDeletedPayload{
					RoundID:        123,
					EventMessageID: "message789",
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						deletedPayload := p.(*roundevents.RoundDeletedPayload)
						deletedPayload.RoundID = payload.RoundID
						deletedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetDeleteRoundManager().
					Return(mockDeleteRoundManager).
					Times(1)

				mockDeleteRoundManager.EXPECT().
					DeleteEmbed(gomock.Any(), roundtypes.EventMessageID("message789"), "channel123").
					Return(true, nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":   payload.RoundID,
					"event_type": "round_deleted",
					"status":     "embed_deleted",
					"message_id": payload.EventMessageID,
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
				Config:       tt.fields.Config,
				Helpers:      tt.fields.Helpers,
				RoundDiscord: tt.fields.RoundDiscord,
			}
			got, err := h.HandleRoundDeleted(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleRoundDeleted() = %v, want nil", got)
				}
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleRoundDeleted() returned %d messages, want %d", len(got), len(tt.want))
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
