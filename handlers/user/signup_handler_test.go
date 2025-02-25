package userhandlers

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockDiscord := mocks.NewMockOperations(ctrl)
	mockLogger := observability.NewNoOpLogger()
	tagNumber := 456

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
			name: "successful user signup request",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": "456"}`)),
			},
			want: []*message.Message{
				message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": "456"}`)),
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
					Return(message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": "456"}`)), nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
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
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": "456"}`)),
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
			h := &UserHandlers{
				Logger:  tt.fields.Logger,
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
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

func TestUserHandlers_HandleUserCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"rattler": "rattler_role_id",
			},
		},
	}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockDiscord := mocks.NewMockOperations(ctrl)
	mockLogger := observability.NewNoOpLogger()

	tagNumber := 456

	type fields struct {
		Logger  observability.Logger
		Config  *config.Config
		Helper  *util_mocks.MockHelpers
		Discord *mocks.MockOperations
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
			name: "successful user created",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": 456}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "123", "rattler_role_id").
					Return(nil).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Signup complete! Your tag number is 456. You now have access to the members-only channels.").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
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
			name: "missing interaction token or guild ID",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": 456}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "no Discord role mapping found",
			fields: fields{
				Logger:  mockLogger,
				Config:  &config.Config{},
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": 456}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Failed to update role: no Discord role mapping found for application role: rattler").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to add Discord role",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": 456}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "123", "rattler_role_id").
					Return(errors.New("add role error")).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Signup succeeded, but failed to sync Discord role: add role error. Contact an admin").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to edit interaction response",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"discord_id": "123", "tag_number": 456}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreatedPayload{
					DiscordID: "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "123", "rattler_role_id").
					Return(nil).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Signup complete! Your tag number is 456. You now have access to the members-only channels.").
					Return(errors.New("edit interaction response error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if tt.name != "missing interaction token or guild ID" {
					tt.args.msg.Metadata.Set("interaction_id", "interaction_id")
					tt.args.msg.Metadata.Set("interaction_token", "interaction_token")
					tt.args.msg.Metadata.Set("guild_id", "guild_123")
				}
				tt.setup()
			}
			h := &UserHandlers{
				Logger:  tt.fields.Logger,
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
			}
			got, err := h.HandleUserCreated(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserHandlers.HandleUserCreated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHandlers_HandleUserCreationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := &config.Config{}
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockDiscord := mocks.NewMockOperations(ctrl)
	mockLogger := observability.NewNoOpLogger()

	type fields struct {
		Logger  observability.Logger
		Config  *config.Config
		Helper  utils.Helpers
		Discord discord.Operations
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
			name: "successful handling of user creation failure",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "some reason"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "some reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Signup failed: some reason. Please try again by reacting to the message in the signup channel, or contact an administrator.").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
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
			name: "missing interaction token in metadata",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "some reason"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "some reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "failed to edit interaction response",
			fields: fields{
				Logger:  mockLogger,
				Config:  mockConfig,
				Helper:  mockHelper,
				Discord: mockDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "some reason"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "some reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockDiscord.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "interaction_token", "Signup failed: some reason. Please try again by reacting to the message in the signup channel, or contact an administrator.").
					Return(errors.New("edit interaction response error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if tt.name != "missing interaction token in metadata" {
					tt.args.msg.Metadata.Set("interaction_token", "interaction_token")
				}
				tt.setup()
			}
			h := &UserHandlers{
				Logger:  tt.fields.Logger,
				Config:  tt.fields.Config,
				Helper:  tt.fields.Helper,
				Discord: tt.fields.Discord,
			}
			got, err := h.HandleUserCreationFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserHandlers.HandleUserCreationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
