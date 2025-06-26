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
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
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

func Test_userHandlers_HandleUserSignupRequest(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful user signup request",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`))
			}(),
			want: []*message.Message{
				func() *message.Message {
					return message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`))
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				tagNumber := sharedtypes.TagNumber(456)
				expectedPayload := userevents.UserSignupRequestPayload{
					UserID:    "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserSignupRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserSignupRequestPayload) = expectedPayload
						return nil
					})
				expectedOutMsg := func() *message.Message {
					return message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`))
				}()
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(expectedPayload), gomock.Eq(userevents.UserSignupRequest)).
					Return(expectedOutMsg, nil)
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
			name: "failed to create backend event",
			msg: func() *message.Message {
				return message.NewMessage("1", []byte(`{"user_id": "123", "tag_number": "456"}`))
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				tagNumber := sharedtypes.TagNumber(456)
				expectedPayload := userevents.UserSignupRequestPayload{
					UserID:    "123",
					TagNumber: &tagNumber,
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserSignupRequestPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserSignupRequestPayload) = expectedPayload
						return nil
					})
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(expectedPayload), gomock.Eq(userevents.UserSignupRequest)).
					Return(nil, errors.New("create backend event error"))
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

			got, err := h.HandleUserSignupRequest(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
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

func Test_userHandlers_HandleUserCreated(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful user created event",
			msg:  func() *message.Message { return message.NewMessage("1", []byte(`{"discord_id": "123"}`)) }(),
			want: []*message.Message{
				func() *message.Message {
					return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "registered_role_id"}`))
				}(),
			},
			wantErr: false,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := userevents.UserCreatedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					})
				rolePayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "registered_role_id",
				}
				expectedOutMsg := func() *message.Message {
					return message.NewMessage("1", []byte(`{"discord_id": "123", "role_id": "registered_role_id"}`))
				}()
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(rolePayload), gomock.Eq(discorduserevents.SignupAddRole)).
					Return(expectedOutMsg, nil)
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
			name: "failed to create add role event",
			msg:  func() *message.Message { return message.NewMessage("1", []byte(`{"discord_id": "123"}`)) }(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := userevents.UserCreatedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreatedPayload) = expectedPayload
						return nil
					})
				rolePayload := discorduserevents.AddRolePayload{
					UserID: "123",
					RoleID: "registered_role_id",
				}
				mockHelper.EXPECT().
					CreateResultMessage(gomock.Eq(msg), gomock.AssignableToTypeOf(rolePayload), gomock.Eq(discorduserevents.SignupAddRole)).
					Return(nil, errors.New("create add role event error"))
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
						RegisteredRoleID: "registered_role_id",
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

			got, err := h.HandleUserCreated(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserCreated() error = %v, wantErr %v", err, tt.wantErr)
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

func Test_userHandlers_HandleUserCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful user creation failed event",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"reason": "test reason"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "test reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					})
				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", false).Return(signup.SignupOperationResult{}, nil) // Assuming signup.SignupOperationResult is compatible; if not, we'd need signup.SignupOperationResult{}
			},
		},
		{
			name: "failed to unmarshal payload",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`invalid payload`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Eq(msg), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to send signup failure response",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"reason": "test reason"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := userevents.UserCreationFailedPayload{
					Reason: "test reason",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&userevents.UserCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*userevents.UserCreationFailedPayload) = expectedPayload
						return nil
					})
				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", false).Return(signup.SignupOperationResult{}, errors.New("send error")) // Assuming signup.SignupOperationResult is compatible
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

			got, err := h.HandleUserCreationFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleUserCreationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userHandlers_HandleRoleAdded(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful role added event",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"discord_id": "123"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.RoleAddedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.RoleAddedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAddedPayload) = expectedPayload
						return nil
					})

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", true).Return(signup.SignupOperationResult{}, nil) // Corrected Return
			},
		},
		{
			name: "failed to unmarshal payload",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`invalid payload`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Eq(msg), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to send signup success response",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"discord_id": "123"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.RoleAddedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.RoleAddedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAddedPayload) = expectedPayload
						return nil
					})

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", true).Return(signup.SignupOperationResult{}, errors.New("send error")) // Corrected Return
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

			got, err := h.HandleRoleAdded(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoleAdded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoleAdded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userHandlers_HandleRoleAdditionFailed(t *testing.T) {
	tests := []struct {
		name    string
		msg     *message.Message
		want    []*message.Message
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockUserDiscordInterface, *util_mocks.MockHelpers, *message.Message)
	}{
		{
			name: "successful role addition failed event",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"discord_id": "123"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: false,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.RoleAdditionFailedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.RoleAdditionFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAdditionFailedPayload) = expectedPayload
						return nil
					})

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", false).Return(signup.SignupOperationResult{}, nil) // Corrected Return type
			},
		},
		{
			name: "failed to unmarshal payload",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`invalid payload`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, _ *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				mockHelper.EXPECT().UnmarshalPayload(gomock.Eq(msg), gomock.Any()).Return(errors.New("unmarshal error"))
			},
		},
		{
			name: "failed to send signup failure response",
			msg: func() *message.Message {
				m := message.NewMessage("1", []byte(`{"discord_id": "123"}`))
				m.Metadata.Set("correlation_id", "correlation_id")
				return m
			}(),
			want: nil, wantErr: true,
			setup: func(ctrl *gomock.Controller, mockUserDiscord *mocks.MockUserDiscordInterface, mockHelper *util_mocks.MockHelpers, msg *message.Message) {
				expectedPayload := discorduserevents.RoleAdditionFailedPayload{
					UserID: "123",
				}
				mockHelper.EXPECT().
					UnmarshalPayload(gomock.Eq(msg), gomock.AssignableToTypeOf(&discorduserevents.RoleAdditionFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discorduserevents.RoleAdditionFailedPayload) = expectedPayload
						return nil
					})

				mockSignupManager := mocks.NewMockSignupManager(ctrl)
				mockUserDiscord.EXPECT().GetSignupManager().Return(mockSignupManager)
				mockSignupManager.EXPECT().SendSignupResult(gomock.Any(), "correlation_id", false).Return(signup.SignupOperationResult{}, errors.New("send error")) // Corrected Return type
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

			got, err := h.HandleRoleAdditionFailed(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoleAdditionFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoleAdditionFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
