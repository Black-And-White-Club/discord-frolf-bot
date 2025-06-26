package userhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_userHandlers_HandleRoleUpdateResult(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful role update result",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "admin", "success": true, "error": ""}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"guild_id":          "guild_123",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "admin",
					Success: true,
					Error:   "",
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
					AddRoleToUser(gomock.Any(), "guild_123", sharedtypes.DiscordID("123"), "discord_admin_role_id").
					Return(role.RoleOperationResult{}, nil).
					Times(1)

				mockRoleManager.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Role update completed").
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`invalid payload`),
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "role mapping not found",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "unknown", "success": true, "error": ""}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"guild_id":          "guild_123",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "unknown",
					Success: true,
					Error:   "",
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
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Failed to update role: no Discord role mapping found for application role: unknown").
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "role update failed",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "admin", "success": false, "error": "discord error"}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"guild_id":          "guild_123",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "admin",
					Success: false,
					Error:   "discord error",
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
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Failed to update role: discord error").
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "guild ID missing",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "admin", "success": true, "error": ""}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "admin",
					Success: true,
					Error:   "",
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
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Failed to update role: guild ID missing from message metadata").
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "failed to add role in Discord",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "admin", "success": true, "error": ""}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"guild_id":          "guild_123",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "admin",
					Success: true,
					Error:   "",
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
					AddRoleToUser(gomock.Any(), "guild_123", sharedtypes.DiscordID("123"), "discord_admin_role_id").
					Return(role.RoleOperationResult{}, errors.New("discord add role error")).
					Times(1)

				mockRoleManager.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Role updated in application, but failed to sync with Discord: discord add role error").
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "failed to send final ephemeral message",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"discord_id": "123", "role": "admin", "success": true, "error": ""}`),
				Metadata: message.Metadata{
					"interaction_token": " interaction_token",
					"guild_id":          "guild_123",
					"correlation_id":    "correlation_id",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				expectedPayload := userevents.UserRoleUpdateResultPayload{
					UserID:  "123",
					Role:    "admin",
					Success: true,
					Error:   "",
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
					AddRoleToUser(gomock.Any(), "guild_123", sharedtypes.DiscordID("123"), "discord_admin_role_id").
					Return(role.RoleOperationResult{}, nil).
					Times(1)

				mockRoleManager.EXPECT().
					EditRoleUpdateResponse(gomock.Any(), "correlation_id", "Role update completed").
					Return(role.RoleOperationResult{}, errors.New("ephemeral send error")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHelper := util_mocks.NewMockHelpers(ctrl)
			mockUserDiscord := mocks.NewMockUserDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockMetrics := &discordmetrics.NoOpMetrics{}
			mockTracer := noop.NewTracerProvider().Tracer("test")

			tt.setup(ctrl, mockUserDiscord, mockHelper)

			h := &UserHandlers{
				Logger: mockLogger,
				Config: &config.Config{
					Discord: config.DiscordConfig{
						RoleMappings: map[string]string{
							"admin": "discord_admin_role_id",
						},
					},
				},
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Tracer:      mockTracer,
				Metrics:     mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleRoleUpdateResult(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoleUpdateResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoleUpdateResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
