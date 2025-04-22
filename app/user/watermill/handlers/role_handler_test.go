package userhandlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_userHandlers_HandleRoleUpdateCommand(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers)
	}{
		{
			name: "successful_role_update_result",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"target_user_id": "456"}`),
				Metadata: message.Metadata{
					"interaction_id":    "interaction_id",
					"interaction_token": "interaction_token",
					"guild_id":          "guild_123",
				},
			},
			want:    nil,
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
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
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).AnyTimes()

				mockRoleManager.EXPECT().
					RespondToRoleRequest(gomock.Any(), "interaction_id", "interaction_token", sharedtypes.DiscordID("456")).
					Return(role.RoleOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "failed_to_unmarshal_payload",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`invalid payload`),
				Metadata: message.Metadata{
					"interaction_id":    "interaction_id",
					"interaction_token": "interaction_token",
					"guild_id":          "guild_123",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed_to_respond_to_role_request",
			msg: &message.Message{
				UUID:    "1",
				Payload: []byte(`{"target_user_id": "456"}`),
				Metadata: message.Metadata{
					"interaction_id":    "interaction_id",
					"interaction_token": "interaction_token",
					"guild_id":          "guild_123",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers) {
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
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager).AnyTimes()

				mockRoleManager.EXPECT().
					RespondToRoleRequest(gomock.Any(), "interaction_id", "interaction_token", sharedtypes.DiscordID("456")).
					Return(role.RoleOperationResult{}, errors.New("failed to send response")).
					Times(1)
			},
		},
		{
			name: "missing_interaction_metadata",
			msg: &message.Message{
				UUID:     "1",
				Payload:  []byte(`{"target_user_id": "456"}`),
				Metadata: message.Metadata{}, // Missing interaction metadata
			},
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, _ *util_mocks.MockHelpers) {
				// No mock expectations needed as the error occurs before interacting with mocks
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
				Logger:      mockLogger,
				Config:      &config.Config{}, // Provide a non-nil config
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Tracer:      mockTracer,
				Metrics:     mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleRoleUpdateCommand(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoleUpdateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoleUpdateCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userHandlers_HandleRoleUpdateButtonPress(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful role update button press",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{
					"interaction_id": "123",
					"interaction_token": "token",
					"requester_id": "456",
					"target_user_id": "789",
					"interaction_custom_id": "role_button_admin",
					"guild_id": "guild_123"
				}`))
			}(),
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("1", []byte(`{"requester_id": "456", "user_id": "789", "role": "admin"}`))
					msg.Metadata.Set("interaction_token", "token")
					msg.Metadata.Set("guild_id", "guild_123")
					return msg
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.RoleUpdateButtonPressPayload{
					InteractionID:       "123",
					InteractionToken:    "token",
					RequesterID:         "456",
					TargetUserID:        "789",
					InteractionCustomID: "role_button_admin",
					GuildID:             "guild_123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.RoleUpdateButtonPressPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleUpdateButtonPressPayload) = expectedPayload
						return nil
					})

				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", sharedtypes.DiscordID("456"), "admin", sharedtypes.DiscordID("789")).
					Return(role.RoleOperationResult{}, nil)

				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)

				backendPayload := userevents.UserRoleUpdateRequestPayload{
					RequesterID: sharedtypes.DiscordID("456"),
					UserID:      sharedtypes.DiscordID("789"),
					Role:        sharedtypes.UserRoleEnum("admin"),
				}

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(backendPayload), gomock.Eq(userevents.UserRoleUpdateRequest)).
					DoAndReturn(func(_ *message.Message, _ any, _ string) (*message.Message, error) {
						outMsg := message.NewMessage("1", []byte(`{"requester_id": "456", "user_id": "789", "role": "admin"}`))
						outMsg.Metadata.Set("interaction_token", "token")
						outMsg.Metadata.Set("guild_id", "guild_123")
						return outMsg, nil
					})
			},
		},
		{
			name: "failed to unmarshal payload",
			msg:  message.NewMessage("1", []byte(`invalid payload`)),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to acknowledge interaction",
			msg:  message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
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
					})

				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", sharedtypes.DiscordID("456"), "admin", sharedtypes.DiscordID("789")).
					Return(role.RoleOperationResult{}, errors.New("acknowledge error")) // Corrected return values

				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)
			},
		},
		{
			name: "failed to create result message",
			msg:  message.NewMessage("1", []byte(`{"interaction_id": "123", "interaction_token": "token", "requester_id": "456", "target_user_id": "789", "interaction_custom_id": "role_button_admin", "guild_id": "guild_123"}`)),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
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
					})

				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockRoleManager.EXPECT().
					RespondToRoleButtonPress(gomock.Any(), "123", "token", sharedtypes.DiscordID("456"), "admin", sharedtypes.DiscordID("789")).
					Return(role.RoleOperationResult{}, nil) // Corrected return values

				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)

				mockHelper.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create result message error"))
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

			if tt.setup != nil {
				tt.setup(ctrl, mockUserDiscord, mockHelper, tt.msg)
			}

			h := &UserHandlers{
				Config:      &config.Config{},
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
				Tracer:      mockTracer,
				Metrics:     mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleRoleUpdateButtonPress(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoleUpdateButtonPress() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.want) == 0 && len(got) == 0 {
				return // no messages expected or returned — ✅
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unexpected number of messages: got %d, want %d", len(got), len(tt.want))
			}

			if len(got) > 0 && len(tt.want) > 0 {
				if !bytes.Equal(got[0].Payload, tt.want[0].Payload) {
					t.Errorf("Payload mismatch.\nGot:  %s\nWant: %s", got[0].Payload, tt.want[0].Payload)
				}

				if diff := cmp.Diff(got[0].Metadata, tt.want[0].Metadata); diff != "" {
					t.Errorf("Metadata mismatch (-got +want):\n%s", diff)
				}
			}
		})
	}
}

func Test_userHandlers_HandleAddRole(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful add role event",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`))
			}(),
			want: []*message.Message{
				func() *message.Message { return message.NewMessage("1", []byte(`{"discord_id": "123"}`)) }(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					})
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", sharedtypes.DiscordID("123"), "role_id").Return(role.RoleOperationResult{}, nil)
				successPayload := discorduserevents.RoleAddedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(successPayload), gomock.Eq(discorduserevents.SignupRoleAdded)).
					DoAndReturn(func(_ *message.Message, _ any, _ string) (*message.Message, error) {
						return message.NewMessage("1", []byte(`{"discord_id": "123"}`)), nil
					})
			},
		},
		{
			name: "failed to unmarshal payload",
			msg:  func() *message.Message { return message.NewMessage("1", []byte(`invalid payload`)) }(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Eq(msg), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to add role",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`))
			}(),
			want: []*message.Message{
				func() *message.Message {
					return message.NewMessage("1", []byte(`{"discord_id": "123", "reason": "add role error"}`))
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					})
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", sharedtypes.DiscordID("123"), "role_id").Return(role.RoleOperationResult{}, errors.New("add role error"))
				failurePayload := discorduserevents.RoleAdditionFailedPayload{
					UserID: "123",
					Reason: "add role error",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(failurePayload), gomock.Eq(discorduserevents.SignupRoleAdditionFailed)).
					DoAndReturn(func(_ *message.Message, _ any, _ string) (*message.Message, error) {
						return message.NewMessage("1", []byte(`{"discord_id": "123", "reason": "add role error"}`)), nil
					})
			},
		},
		{
			name: "failed to create success message",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`))
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					})
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", sharedtypes.DiscordID("123"), "role_id").Return(role.RoleOperationResult{}, nil)
				successPayload := discorduserevents.RoleAddedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(successPayload), gomock.Eq(discorduserevents.SignupRoleAdded)).
					Return(nil, errors.New("create success message error"))
			},
		},
		{
			name: "failed to create failure message",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "role_id"}`))
			}(),
			want:    nil,
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "role_id",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.AddRolePayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.AddRolePayload) = expectedPayload
						return nil
					})
				mockRoleManager := mocks.NewMockRoleManager(ctrl)
				mockUserDiscord.EXPECT().GetRoleManager().Return(mockRoleManager)
				mockRoleManager.EXPECT().AddRoleToUser(gomock.Any(), "guild_id", sharedtypes.DiscordID("123"), "role_id").Return(role.RoleOperationResult{}, errors.New("add role error"))
				failurePayload := discorduserevents.RoleAdditionFailedPayload{
					UserID: "123",
					Reason: "add role error",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(failurePayload), gomock.Eq(discorduserevents.SignupRoleAdditionFailed)).
					Return(nil, errors.New("create failure message error"))
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

			if tt.setup != nil {
				tt.setup(ctrl, mockUserDiscord, mockHelper, tt.msg)
			}

			h := &UserHandlers{
				Config: &config.Config{
					Discord: config.DiscordConfig{
						GuildID: "guild_id",
					},
				},
				Helper:      mockHelper,
				UserDiscord: mockUserDiscord,
				Logger:      mockLogger,
				Tracer:      mockTracer,
				Metrics:     mockMetrics,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return wrapHandler(handlerName, unmarshalTo, handlerFunc, mockLogger, mockMetrics, mockTracer, mockHelper)
				},
			}

			got, err := h.HandleAddRole(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleAddRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) == 0 && len(got) == 0 {
				return // no messages expected or returned — ✅
			}

			if len(got) != len(tt.want) {
				t.Fatalf("unexpected number of messages: got %d, want %d", len(got), len(tt.want))
			}

			if len(got) > 0 && len(tt.want) > 0 {
				if !bytes.Equal(got[0].Payload, tt.want[0].Payload) {
					t.Errorf("Payload mismatch.\nGot:  %s\nWant: %s", got[0].Payload, tt.want[0].Payload)
				}

				if diff := cmp.Diff(got[0].Metadata, tt.want[0].Metadata); diff != "" {
					t.Errorf("Metadata mismatch (-got +want):\n%s", diff)
				}
			}
		})
	}
}
