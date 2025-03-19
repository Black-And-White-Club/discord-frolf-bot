package roundhandlers

import (
	"fmt"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantScoreUpdated_EdgeCases(t *testing.T) {
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
		wantErr bool
		setup   func()
	}{
		{
			name: "unmarshal error",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid json`)),
			},
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("invalid JSON")).
					Times(1)
			},
		},
		{
			name: "negative score value",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"participant": "user-123",
					"score": -5,
					"channel_id": "channel-123",
					"message_id": "message-456"
				}`)),
			},
			wantErr: false,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *message.Message, p interface{}) error {
						payload := p.(*roundevents.ParticipantScoreUpdatedPayload)
						score := -5 // Negative score
						payload.RoundID = 123
						payload.Participant = "user-123"
						payload.Score = score
						payload.ChannelID = "channel-123"
						payload.MessageID = "message-456"
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateConfirmation(
						gomock.Eq("channel-123"),
						gomock.Eq(roundtypes.UserID("user-123")),
						gomock.Any(),
					).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "discord error on confirmation send",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"round_id": 123,
					"participant": "user-123",
					"score": 42,
					"channel_id": "channel-123",
					"message_id": "message-456"
				}`)),
			},
			wantErr: false, // Should not return error even if Discord fails
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *message.Message, p interface{}) error {
						payload := p.(*roundevents.ParticipantScoreUpdatedPayload)
						score := 42
						payload.RoundID = 123
						payload.Participant = "user-123"
						payload.Score = score
						payload.ChannelID = "channel-123"
						payload.MessageID = "message-456"
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateConfirmation(
						gomock.Eq("channel-123"),
						gomock.Eq(roundtypes.UserID("user-123")),
						gomock.Any(),
					).
					Return(fmt.Errorf("discord API error")).
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

			_, err := h.HandleParticipantScoreUpdated(tt.args.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateError_EdgeCases(t *testing.T) {
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
		wantErr bool
		setup   func()
	}{
		{
			name: "unmarshal error",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid json`)),
			},
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("invalid JSON")).
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
					"score_update_request": {
						"round_id": 123,
						"participant": "user-456",
						"score": 42
					},
					"error": ""
				}`)),
			},
			wantErr: false,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *message.Message, p interface{}) error {
						payload := p.(*roundevents.RoundScoreUpdateErrorPayload)

						payload.ScoreUpdateRequest = &roundevents.ScoreUpdateRequestPayload{
							RoundID:     123,
							Participant: roundtypes.UserID("user-456"),
						}

						score := 42
						payload.ScoreUpdateRequest.Score = &score

						// Empty error message
						payload.Error = ""
						return nil
					}).
					Times(1)

				// No further calls expected when error is empty
			},
		},
		{
			name: "discord error on sending error message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"score_update_request": {
						"round_id": 123,
						"participant": "user-456",
						"score": 42
					},
					"error": "Invalid score format"
				}`)),
			},
			wantErr: false, // Should not return error even if Discord fails
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *message.Message, p interface{}) error {
						payload := p.(*roundevents.RoundScoreUpdateErrorPayload)

						payload.ScoreUpdateRequest = &roundevents.ScoreUpdateRequestPayload{
							RoundID:     123,
							Participant: roundtypes.UserID("user-456"),
						}

						score := 42
						payload.ScoreUpdateRequest.Score = &score
						payload.Error = "Invalid score format"
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateError(
						gomock.Eq(roundtypes.UserID("user-456")),
						gomock.Eq("Invalid score format"),
					).
					Return(fmt.Errorf("discord API error")).
					Times(1)
			},
		},
		{
			name: "nil score in request",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{
					"score_update_request": {
						"round_id": 123,
						"participant": "user-456",
						"score": null
					},
					"error": "Score cannot be null"
				}`)),
			},
			wantErr: false,
			setup: func() {
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ *message.Message, p interface{}) error {
						payload := p.(*roundevents.RoundScoreUpdateErrorPayload)

						payload.ScoreUpdateRequest = &roundevents.ScoreUpdateRequestPayload{
							RoundID:     123,
							Participant: roundtypes.UserID("user-456"),
							Score:       nil, // Nil score
						}

						payload.Error = "Score cannot be null"
						return nil
					}).
					Times(1)

				mockRoundDiscord.EXPECT().
					GetScoreRoundManager().
					Return(mockScoreRoundManager).
					Times(1)

				mockScoreRoundManager.EXPECT().
					SendScoreUpdateError(
						gomock.Eq(roundtypes.UserID("user-456")),
						gomock.Eq("Score cannot be null"),
					).
					Return(nil).
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

			_, err := h.HandleScoreUpdateError(tt.args.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
