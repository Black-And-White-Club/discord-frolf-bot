package userhandlers

import (
	"errors"
	"testing"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func Test_userHandlers_HandleRoleUpdateCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)
	type fields struct {
		Config      *config.Config
		Helper      *util_mocks.MockHelpers
		UserDiscord *mocks.MockUserDiscordInterface
		Logger      observability.Logger
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
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
			},
			args: args{
				msg: &message.Message{
					UUID:    "1",
					Payload: []byte(`{"target_user_id": "456", "role": "admin"}`),
					Metadata: message.Metadata{
						"interaction_id":    "interaction_id",
						"interaction_token": "interaction_token",
						"guild_id":          "guild_123",
					},
				},
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateCommandPayload{
					TargetUserID: "456",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateCommandPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateCommandPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)

				mockRoleManager.EXPECT().
					RespondToRoleRequest(gomock.Any(), "interaction_id", "interaction_token", "456").
					Return(nil).
					Times(1)

				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1) // Update to Times(2)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mocks.NewMockSignupManager(ctrl)).AnyTimes()
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
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
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"target_user_id": "456", "role": "admin"}`)), // Corrected payload
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.RoleUpdateCommandPayload{ // Correct payload type
					TargetUserID: "456",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateCommandPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateCommandPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mocks.NewMockSignupManager(ctrl)).AnyTimes()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			h := &userHandlers{
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
				Logger:      tt.fields.Logger,
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

func Test_userHandlers_HandleRoleUpdateButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()
	type fields struct {
		Logger      observability.Logger
		Config      *config.Config
		Helper      *util_mocks.MockHelpers
		UserDiscord *mocks.MockUserDiscordInterface
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
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"requester_id": "456", "user_id": "789", "role": "admin"}`)),
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

				// Create RoleManager mock
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(nil).
					Times(1)

				// Configure UserDiscord mock
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mocks.NewMockSignupManager(ctrl)).AnyTimes()

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(message.NewMessage("1", []byte(`{"requester_id": "456", "user_id": "789", "role": "admin"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
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
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
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

				// Create RoleManager mock
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(errors.New("acknowledge error")).
					Times(1)

				// Configure UserDiscord mock
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mocks.NewMockSignupManager(ctrl)).AnyTimes()
			},
		},
		{
			name: "failed to create result message",
			fields: fields{
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
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

				// Create RoleManager mock
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", "456", "admin", "789").
					Return(nil).
					Times(1)

				// Configure UserDiscord mock
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mocks.NewMockSignupManager(ctrl)).AnyTimes()

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
			h := &userHandlers{
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
				Logger:      tt.fields.Logger,
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

func Test_userHandlers_HandleAddRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild_id",
		},
	}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()
	type fields struct {
		Logger      observability.Logger
		Config      *config.Config
		EventUtil   utils.EventUtil
		Helper      *util_mocks.MockHelpers
		UserDiscord *mocks.MockUserDiscordInterface
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
			name: "successful add role event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"discord_id": "123"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.AddRolePayload{
					DiscordID: "123",
					RoleID:    "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", "123", "role_id").Return(nil).Times(1)
				successPayload := discorduserevents.RoleAddedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), successPayload, discorduserevents.SignupRoleAdded).
					Return(message.NewMessage("1", []byte(`{"discord_id": "123"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to add role",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"discord_id": "123", "reason": "add role error"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.AddRolePayload{
					DiscordID: "123",
					RoleID:    "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", "123", "role_id").Return(errors.New("add role error")).Times(1)
				failurePayload := discorduserevents.RoleAdditionFailedPayload{
					DiscordID: "123",
					Reason:    "add role error",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), failurePayload, discorduserevents.SignupRoleAdditionFailed).
					Return(message.NewMessage("1", []byte(`{"discord_id": "123", "reason": "add role error"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to create success message",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.AddRolePayload{
					DiscordID: "123",
					RoleID:    "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", "123", "role_id").Return(nil).Times(1)
				successPayload := discorduserevents.RoleAddedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), successPayload, discorduserevents.SignupRoleAdded).
					Return(nil, errors.New("create success message error")).
					Times(1)
			},
		},
		{
			name: "failed to create failure message",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discorduserevents.AddRolePayload{
					DiscordID: "123",
					RoleID:    "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).Times(1)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", "123", "role_id").Return(errors.New("add role error")).Times(1)
				failurePayload := discorduserevents.RoleAdditionFailedPayload{
					DiscordID: "123",
					Reason:    "add role error",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), failurePayload, discorduserevents.SignupRoleAdditionFailed).
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
			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}
			got, err := h.HandleAddRole(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleAddRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("userHandlers.HandleAddRole() returned %d messages, want %d", len(got), len(tt.want))
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
