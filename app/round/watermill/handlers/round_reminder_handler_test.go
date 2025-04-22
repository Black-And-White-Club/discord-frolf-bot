package roundhandlers

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockRoundReminderManager := mocks.NewMockRoundReminderManager(ctrl)

	type fields struct {
		Logger       *slog.Logger
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
			name: "successful reminder",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"round_title": "Test Round",
					"user_ids": ["user1", "user2"],
					"reminder_type": "1-hour",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
			}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"reminder_type":"1-hour","status":"reminder_sent"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				payload := &roundevents.DiscordReminderPayload{
					RoundID:          123,
					RoundTitle:       "Test Round",
					UserIDs:          []roundtypes.UserID{"user1", "user2"},
					ReminderType:     "1-hour",
					DiscordChannelID: "channel123",
					DiscordGuildID:   "guild456",
					EventMessageID:   "message789",
				}

				mockRoundDiscord.EXPECT().
					GetRoundReminderManager().
					Return(mockRoundReminderManager).
					Times(1)

				mockRoundReminderManager.EXPECT().
					SendRoundReminder(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx interface{}, p *roundevents.DiscordReminderPayload) (bool, error) {
						if p.RoundID != payload.RoundID || p.ReminderType != payload.ReminderType {
							t.Errorf("Expected payload %v, got %v", payload, p)
						}
						return true, nil
					}).
					Times(1)

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"reminder_type":"1-hour","status":"reminder_sent"}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
					Return(traceMsg, nil).
					Times(1)
			},
		},
		{
			name: "reminder processing failed but no error",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"round_title": "Test Round",
					"user_ids": ["user1", "user2"],
					"reminder_type": "1-hour",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"reminder_type":"1-hour","status":"reminder_failed"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				mockRoundDiscord.EXPECT().
					GetRoundReminderManager().
					Return(mockRoundReminderManager).
					Times(1)

				mockRoundReminderManager.EXPECT().
					SendRoundReminder(gomock.Any(), gomock.Any()).
					Return(false, nil).
					Times(1)

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"reminder_type":"1-hour","status":"reminder_failed"}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
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
			setup:   func() {},
		},
		{
			name: "failed to send round reminder",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"round_title": "Test Round",
					"user_ids": ["user1", "user2"],
					"reminder_type": "1-hour",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockRoundDiscord.EXPECT().
					GetRoundReminderManager().
					Return(mockRoundReminderManager).
					Times(1)

				mockRoundReminderManager.EXPECT().
					SendRoundReminder(gomock.Any(), gomock.Any()).
					Return(false, errors.New("failed to send reminder")).
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
					"round_title": "Test Round",
					"user_ids": ["user1", "user2"],
					"reminder_type": "1-hour",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockRoundDiscord.EXPECT().
					GetRoundReminderManager().
					Return(mockRoundReminderManager).
					Times(1)

				mockRoundReminderManager.EXPECT().
					SendRoundReminder(gomock.Any(), gomock.Any()).
					Return(true, nil).
					Times(1)

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundTraceEvent).
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
			got, err := h.HandleRoundReminder(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleRoundReminder() = %v, want nil", got)
				}
			} else if got == nil {
				t.Errorf("RoundHandlers.HandleRoundReminder() = nil, want %v", tt.want)
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleRoundReminder() returned %d messages, want %d", len(got), len(tt.want))
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
