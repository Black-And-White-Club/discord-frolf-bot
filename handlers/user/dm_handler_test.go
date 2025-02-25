package userhandlers

import (
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleSendUserDM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockDiscord := mocks.NewMockOperations(ctrl)
	mockLogger := observability.NewNoOpLogger()

	type fields struct {
		Config  *config.Config
		Helper  *util_mocks.MockHelpers
		Discord *mocks.MockOperations
		Logger  observability.Logger
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
			name: "successful DM",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "message": "Hello"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"user_id": "123", "status": "success"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.SendUserDMPayload{
					UserID:  "123",
					Message: "Hello",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.SendUserDMPayload{})).
					DoAndReturn(func(_ *message.Message, v interface{}) error {
						*v.(*discorduserevents.SendUserDMPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					SendDM(gomock.Any(), "123", "Hello").
					Return(&discordgo.Message{ID: "1", ChannelID: "1"}, nil).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(message.NewMessage("1", []byte(`{"user_id": "123", "status": "success"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to send DM",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "message": "Hello"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"user_id": "123", "status": "fail"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.SendUserDMPayload{
					UserID:  "123",
					Message: "Hello",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.SendUserDMPayload{})).
					DoAndReturn(func(_ *message.Message, v interface{}) error {
						*v.(*discorduserevents.SendUserDMPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					SendDM(gomock.Any(), "123", "Hello").
					Return(nil, errors.New("send DM error")).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(message.NewMessage("1", []byte(`{"user_id": "123", "status": "fail"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to create success message",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "message": "Hello"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.SendUserDMPayload{
					UserID:  "123",
					Message: "Hello",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.SendUserDMPayload{})).
					DoAndReturn(func(_ *message.Message, v interface{}) error {
						*v.(*discorduserevents.SendUserDMPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					SendDM(gomock.Any(), "123", "Hello").
					Return(&discordgo.Message{ID: "1", ChannelID: "1"}, nil).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create success message error")).
					Times(1)
			},
		},
		{
			name: "failed to create failure message",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "message": "Hello"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.SendUserDMPayload{
					UserID:  "123",
					Message: "Hello",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.SendUserDMPayload{})).
					DoAndReturn(func(_ *message.Message, v interface{}) error {
						*v.(*discorduserevents.SendUserDMPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					SendDM(gomock.Any(), "123", "Hello").
					Return(nil, errors.New("send DM error")).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create failure message error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &UserHandlers{
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
				Logger:  tt.fields.Logger,
			}
			got, err := h.HandleSendUserDM(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleSendUserDM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UserHandlers.HandleSendUserDM() returned %d messages, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].UUID != tt.want[i].UUID || string(got[i].Payload) != string(tt.want[i].Payload) {
					t.Errorf("Message mismatch at index %d:\nGot:  %s\nWant: %s", i, string(got[i].Payload), string(tt.want[i].Payload))
				}
			}
		})
	}
}
