package userhandlers

import (
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleRoleUpdateCommand(t *testing.T) {
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
			name: "successful role update command",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"target_user_id": "123"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateCommandPayload{
					TargetUserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateCommandPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateCommandPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					RespondToRoleRequest(gomock.Any(), "interaction_id", "interaction_token", "123").
					Return(nil).
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
			name: "failed to respond to role request",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"target_user_id": "123"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateCommandPayload{
					TargetUserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateCommandPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateCommandPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					RespondToRoleRequest(gomock.Any(), "interaction_id", "interaction_token", "123").
					Return(errors.New("respond error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.args.msg.Metadata.Set("interaction_id", "interaction_id")
				tt.args.msg.Metadata.Set("interaction_token", "interaction_token")
				tt.args.msg.Metadata.Set("guild_id", "guild_123")
				tt.setup()
			}
			h := &UserHandlers{
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
				Logger:  tt.fields.Logger,
			}
			got, err := h.HandleRoleUpdateCommand(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleRoleUpdateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UserHandlers.HandleRoleUpdateCommand() returned %d messages, want %d", len(got), len(tt.want))
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

func TestUserHandlers_HandleRoleUpdateButtonPress(t *testing.T) {
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
			name: "successful role update button press",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"requester_id": "456", "discord_id": "789", "role": "admin"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateButtonPressPayload{
					InteractionID:       "123",
					InteractionToken:    "token",
					RequesterID:         "456",
					TargetUserID:        "789",
					InteractionCustomID: "role_button_admin",
					GuildID:             "guild_123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateButtonPressPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateButtonPressPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(nil).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(message.NewMessage("1", []byte(`{"requester_id": "456", "discord_id": "789", "role": "admin"}`)), nil).
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
			name: "failed to acknowledge interaction",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateButtonPressPayload{
					InteractionID:       "123",
					InteractionToken:    "token",
					RequesterID:         "456",
					TargetUserID:        "789",
					InteractionCustomID: "role_button_admin",
					GuildID:             "guild_123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateButtonPressPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateButtonPressPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(errors.New("acknowledge error")).
					Times(1)
			},
		},
		{
			name: "failed to create result message",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateButtonPressPayload{
					InteractionID:       "123",
					InteractionToken:    "token",
					RequesterID:         "456",
					TargetUserID:        "789",
					InteractionCustomID: "role_button_admin",
					GuildID:             "guild_123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateButtonPressPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateButtonPressPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(nil).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create result message error")).
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
			got, err := h.HandleRoleUpdateButtonPress(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleRoleUpdateButtonPress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UserHandlers.HandleRoleUpdateButtonPress() returned %d messages, want %d", len(got), len(tt.want))
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

func TestUserHandlers_HandleRoleUpdateResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"admin": "role_admin_id",
			},
		},
	}
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
			name: "successful role update result",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"requester_id": "123", "discord_id": "456", "role": "admin", "success": true}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					DiscordID: usertypes.DiscordID("456"),
					Role:      usertypes.UserRoleEnum("admin"),
					Success:   true,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserRoleUpdateResultPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserRoleUpdateResultPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "456", "role_admin_id").
					Return(nil).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Role update completed").
					Return(nil).
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
			name: "failed to add role to user",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"requester_id": "123", "discord_id": "456", "role": "admin", "success": true}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					DiscordID: usertypes.DiscordID("456"),
					Role:      usertypes.UserRoleEnum("admin"),
					Success:   true,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserRoleUpdateResultPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserRoleUpdateResultPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "456", "role_admin_id").
					Return(errors.New("add role error")).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Role updated in application, but failed to sync with Discord: add role error").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to edit interaction response",
			fields: fields{
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
				Logger:  mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"requester_id": "123", "discord_id": "456", "role": "admin", "success": true}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					DiscordID: usertypes.DiscordID("456"),
					Role:      usertypes.UserRoleEnum("admin"),
					Success:   true,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserRoleUpdateResultPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserRoleUpdateResultPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "456", "role_admin_id").
					Return(nil).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Role update completed").
					Return(errors.New("edit response error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.args.msg.Metadata.Set("interaction_id", "interaction_id")
				tt.args.msg.Metadata.Set("interaction_token", "interaction_token")
				tt.args.msg.Metadata.Set("guild_id", "guild_123")
				tt.setup()
			}
			h := &UserHandlers{
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
				Logger:  tt.fields.Logger,
			}
			got, err := h.HandleRoleUpdateResult(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleRoleUpdateResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UserHandlers.HandleRoleUpdateResult() returned %d messages, want %d", len(got), len(tt.want))
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
