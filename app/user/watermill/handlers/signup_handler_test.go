package userhandlers

import (
	"errors"
	"reflect"
	"testing"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func Test_userHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()
	tagNumber := 456
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
			name: "successful user signup request",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserSignupRequestPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserSignupRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserSignupRequestPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), expectedPayload, userevents.UserSignupRequest).
					Return(message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`)), nil).
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
			wantErr: true,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to create backend event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserSignupRequestPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserSignupRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserSignupRequestPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), expectedPayload, userevents.UserSignupRequest).
					Return(nil, errors.New("create backend event error")).
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
			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}
			got, err := h.HandleUserSignupRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("UserHandlers.HandleUserSignupRequest() returned %d messages, want %d", len(got), len(tt.want))
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

func Test_userHandlers_HandleUserCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RegisteredRoleID: "registered_role_id",
		},
	}
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
			name: "successful user created event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "registered_role_id"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				rolePayload := discorduserevents.AddRolePayload{
					DiscordID: string(expectedPayload.DiscordID),
					RoleID:    "registered_role_id",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), rolePayload, discorduserevents.SignupAddRole).
					Return(message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "registered_role_id"}`)), nil).
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
			wantErr: true,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to create add role event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				rolePayload := discorduserevents.AddRolePayload{
					DiscordID: string(expectedPayload.DiscordID),
					RoleID:    "registered_role_id",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), rolePayload, discorduserevents.SignupAddRole).
					Return(nil, errors.New("create add role event error")).
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
			got, err := h.HandleUserCreated(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleUserCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("userHandlers.HandleUserCreated() returned %d messages, want %d", len(got), len(tt.want))
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

func Test_userHandlers_HandleUserCreationFailed(t *testing.T) {
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
			name: "successful user creation failed event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "test reason"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "test reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager).Times(1)
				mockSignupManager.EXPECT().SendSignupResult("correlation_id", false).Return(nil).Times(1)
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
			wantErr: true,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.args.msg.Metadata.Set("correlation_id", "correlation_id")
				tt.setup()
			}
			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}
			got, err := h.HandleUserCreationFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleUserCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("userHandlers.HandleUserCreationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userHandlers_HandleRoleAdded(t *testing.T) {
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
			name: "successful role added event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.RoleAddedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleAddedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAddedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager).Times(1)
				mockSignupManager.EXPECT().SendSignupResult("correlation_id", true).Return(nil).Times(1)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.args.msg.Metadata.Set("correlation_id", "correlation_id")
				tt.setup()
			}
			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}
			got, err := h.HandleRoleAdded(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleRoleAdded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("userHandlers.HandleRoleAdded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userHandlers_HandleRoleAdditionFailed(t *testing.T) {
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
			name: "successful role addition failed event",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := discorduserevents.RoleAdditionFailedPayload{
					DiscordID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discorduserevents.RoleAdditionFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAdditionFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager).Times(1)
				mockSignupManager.EXPECT().SendSignupResult("correlation_id", false).Return(nil).Times(1)
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
			wantErr: true,
			setup: func() {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.args.msg.Metadata.Set("correlation_id", "correlation_id")
				tt.setup()
			}
			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}
			got, err := h.HandleRoleAdditionFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleRoleAdditionFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("userHandlers.HandleRoleAdditionFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
