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

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockFinalizeRoundManager := mocks.NewMockFinalizeRoundManager(ctrl)

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
			name: "successful round finalized",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"title": "Test Round",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_finalized","status":"scorecard_finalized","message_id":"message789"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				eventMessageID := roundtypes.EventMessageID("message789")
				payload := &roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          123,
					Title:            "Test Round",
					DiscordChannelID: "channel123",
					EventMessageID:   &eventMessageID,
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						finalizedPayload := p.(*roundevents.RoundFinalizedEmbedUpdatePayload)
						finalizedPayload.RoundID = payload.RoundID
						finalizedPayload.Title = payload.Title
						finalizedPayload.DiscordChannelID = payload.DiscordChannelID
						finalizedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					Times(1)

				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), "message789", "channel123", gomock.Any()).
					Return(nil, nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":   payload.RoundID,
					"event_type": "round_finalized",
					"status":     "scorecard_finalized",
					"message_id": payload.EventMessageID,
				}

				traceMsg := message.NewMessage(roundevents.RoundTraceEvent, []byte(`{"round_id":123,"event_type":"round_finalized","status":"scorecard_finalized","message_id":"message789"}`))
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
			name: "missing event message ID",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"title": "Test Round",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				payload := &roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          123,
					Title:            "Test Round",
					DiscordChannelID: "channel123",
					EventMessageID:   nil,
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						finalizedPayload := p.(*roundevents.RoundFinalizedEmbedUpdatePayload)
						finalizedPayload.RoundID = payload.RoundID
						finalizedPayload.Title = payload.Title
						finalizedPayload.DiscordChannelID = payload.DiscordChannelID
						finalizedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "failed to finalize scorecard embed",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"title": "Test Round",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				eventMessageID := roundtypes.EventMessageID("message789")
				payload := &roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          123,
					Title:            "Test Round",
					DiscordChannelID: "channel123",
					EventMessageID:   &eventMessageID,
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						finalizedPayload := p.(*roundevents.RoundFinalizedEmbedUpdatePayload)
						finalizedPayload.RoundID = payload.RoundID
						finalizedPayload.Title = payload.Title
						finalizedPayload.DiscordChannelID = payload.DiscordChannelID
						finalizedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					Times(1)

				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), "message789", "channel123", gomock.Any()).
					Return(nil, errors.New("failed to finalize scorecard embed")).
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
					"title": "Test Round",
					"discord_channel_id": "channel123",
					"discord_guild_id": "guild456",
					"event_message_id": "message789"
				}`)),
			},
			want:    []*message.Message{},
			wantErr: false,
			setup: func() {
				eventMessageID := roundtypes.EventMessageID("message789")
				payload := &roundevents.RoundFinalizedEmbedUpdatePayload{
					RoundID:          123,
					Title:            "Test Round",
					DiscordChannelID: "channel123",
					EventMessageID:   &eventMessageID,
				}

				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(msg *message.Message, p interface{}) error {
						finalizedPayload := p.(*roundevents.RoundFinalizedEmbedUpdatePayload)
						finalizedPayload.RoundID = payload.RoundID
						finalizedPayload.Title = payload.Title
						finalizedPayload.DiscordChannelID = payload.DiscordChannelID
						finalizedPayload.EventMessageID = payload.EventMessageID
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					Times(1)

				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), "message789", "channel123", gomock.Any()).
					Return(nil, nil).
					Times(1)

				tracePayload := map[string]interface{}{
					"round_id":   payload.RoundID,
					"event_type": "round_finalized",
					"status":     "scorecard_finalized",
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
				Helpers:      tt.fields.Helpers,
				RoundDiscord: tt.fields.RoundDiscord,
			}
			got, err := h.HandleRoundFinalized(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleRoundFinalized() = %v, want nil", got)
				}
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleRoundFinalized() returned %d messages, want %d", len(got), len(tt.want))
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
