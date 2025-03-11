package userhandlers

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func Test_userHandlers_HandleRoleUpdateResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			RoleMappings: map[string]string{
				"admin": "discord_admin_role_id",
			},
		},
	}
	mockLogger := observability.NewNoOpLogger()
	mockHelper := util_mocks.NewMockHelpers(ctrl)
	mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)

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
			name: "successful role update result",
			fields: fields{
				Logger:      mockLogger,
				Config:      mockConfig,
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
			},
			args: args{
				msg: &message.Message{
					UUID:    "1",
					Payload: []byte(`{"discord_id": "123", "role": "admin", "success": true, "error": ""}`),
					Metadata: message.Metadata{
						"interaction_token": " interaction_token",
						"guild_id":          "guild_123",
						"correlation_id":    "correlation_id",
					},
				},
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					DiscordID: "123",
					Role:      "admin",
					Success:   true,
					Error:     "",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&userevents.UserRoleUpdateResultPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserRoleUpdateResultPayload) = expectedPayload
						return nil
					}).
					Times(1)
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).AnyTimes()

				mockRoleManager.EXPECT().
					AddRoleToUser(gomock.Any(), "guild_123", "123", "discord_admin_role_id").
					Return(nil).
					Times(1)

				mockRoleManager.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Role update completed").
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

			h := &userHandlers{
				Logger:      tt.fields.Logger,
				Config:      tt.fields.Config,
				Helper:      tt.fields.Helper,
				UserDiscord: tt.fields.UserDiscord,
			}

			got, err := h.HandleRoleUpdateResult(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("userHandlers.HandleRoleUpdateResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("userHandlers.HandleRoleUpdateResult() returned %d messages, want %d", len(got), len(tt.want))
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
