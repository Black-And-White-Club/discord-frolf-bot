package roundhandlers

import (
	"errors"
	"reflect"
	"testing"
	"time"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utils "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundCreateRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)

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
		want    []*message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "successful round create request",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"title": "Test Round", "start_time": "2023-10-10T10:00:00Z"}`)),
			},
			want: func() []*message.Message {
				msg := message.NewMessage(roundevents.RoundCreateRequest, []byte(`{"title":"Test Round","start_time":"2025-03-16T07:42:25-05:00"}`))
				return []*message.Message{msg}
			}(),
			wantErr: false,
			setup: func() {
				expectedPayload := roundevents.CreateRoundRequestedPayload{
					Title:     "Test Round",
					StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.CreateRoundRequestedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.CreateRoundRequestedPayload) = expectedPayload
						return nil
					}).Times(1)

				msg := message.NewMessage(roundevents.RoundCreateRequest, []byte(`{"title":"Test Round","start_time":"2025-03-16T07:42:25-05:00"}`))
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundCreateRequest).
					Return(msg, nil).Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)

				mockRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockRoundManager).
					Times(1)
				mockRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to create result message",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"title": "Test Round", "start_time": "2023-10-10T10:00:00Z"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := roundevents.CreateRoundRequestedPayload{
					Title:     "Test Round",
					StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.CreateRoundRequestedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.CreateRoundRequestedPayload) = expectedPayload
						return nil
					}).Times(1)

				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.Any(), roundevents.RoundCreateRequest).
					Return(nil, errors.New("failed to create result message")).
					Times(1)

				mockRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockRoundManager).
					Times(1)
				mockRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), "Failed to create result message: failed to create result message").
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
			got, err := h.HandleRoundCreateRequested(tt.args.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundCreateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("RoundHandlers.HandleRoundCreateRequested() = %v, want nil", got)
				}
			} else if got == nil {
				t.Errorf("RoundHandlers.HandleRoundCreateRequested() = nil, want %v", tt.want)
			} else if len(got) != len(tt.want) {
				t.Errorf("RoundHandlers.HandleRoundCreateRequested() returned %d messages, want %d", len(got), len(tt.want))
			} else {
				for i, wantMsg := range tt.want {
					if i >= len(got) {
						t.Errorf("Missing expected message at index %d", i)
						continue
					}

					gotMsg := got[i]
					if wantMsg.UUID != gotMsg.UUID {
						t.Errorf("Message UUID mismatch at index %d: got %s, want %s", i, gotMsg.UUID, wantMsg.UUID)
					}

					if string(wantMsg.Payload) != string(gotMsg.Payload) {
						t.Errorf("Message payload mismatch at index %d: got %s, want %s", i, string(gotMsg.Payload), string(wantMsg.Payload))
					}
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

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
			name: "successful round created event",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"round_id": 1, "user_id": "user123", "title": "Test Round", "start_time": "2023-10-10T10:00:00Z"}`)),
			},
			wantErr: false,
			setup: func() {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID: 1,
						Title:   "Test Round",
						UserID:  "user123",
						StartTime: func() *roundtypes.StartTime {
							t, _ := time.Parse(time.RFC3339, "2023-10-10T10:00:00Z")
							startTime := roundtypes.StartTime(t) // Convert time.Time to roundtypes.StartTime
							return &startTime
						}(),
					},
				}

				// Mock payload unmarshal
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).Times(1)

				// Mock `GetCreateRoundManager`
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(2)

				// Mock interaction response update
				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(1)

				// Mock `SendRoundEventEmbed` to return a message with an ID
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), // Channel ID
						gomock.Any(), // Title
						gomock.Any(), // Description
						gomock.Any(), // Start Time
						gomock.Any(), // Location
						gomock.Any(), // Creator ID
						gomock.Any(), // Round ID
					).
					Return(&discordgo.Message{ID: "message-id"}, nil).
					Times(1)

				// Mock event message creation
				mockHelpers.EXPECT().
					CreateResultMessage(gomock.Any(), gomock.AssignableToTypeOf(roundevents.RoundEventMessageIDUpdatedPayload{}), roundevents.RoundEventMessageIDUpdate).
					Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to update interaction response",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"round_id": 1, "user_id": "user123", "title": "Test Round", "start_time": "2023-10-10T10:00:00Z"}`)),
			},
			wantErr: true,
			setup: func() {
				expectedPayload := roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID: 1,
						Title:   "Test Round",
						UserID:  "user123",
						StartTime: func() *roundtypes.StartTime {
							t, _ := time.Parse(time.RFC3339, "2023-10-10T10:00:00Z")
							startTime := roundtypes.StartTime(t) // Convert time.Time to roundtypes.StartTime
							return &startTime
						}(),
					},
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundCreatedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundCreatedPayload) = expectedPayload
						return nil
					}).Times(1)

				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(1)

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponse(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error")).Times(1)
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
			got, err := h.HandleRoundCreated(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if got != nil {
					t.Errorf("HandleRoundCreated() should return nil messages when failing")
				}
			} else {
				if len(got) == 0 {
					t.Errorf("HandleRoundCreated() should return a valid message")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

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
		want    []*message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "successful round creation failed event",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "Some error occurred"}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := discordroundevents.RoundCreationFailedPayload{
					Reason: "Some error occurred",
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundCreationFailedPayload) = expectedPayload
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(1)

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to update interaction response",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"reason": "Some error occurred"}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := discordroundevents.RoundCreationFailedPayload{
					Reason: "Some error occurred",
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&discordroundevents.RoundCreationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*discordroundevents.RoundCreationFailedPayload) = expectedPayload
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(1)

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error")).Times(1)
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
			got, err := h.HandleRoundCreationFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundHandlers.HandleRoundCreationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := observability.NewNoOpLogger()
	mockHelpers := utils.NewMockHelpers(ctrl)
	mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
	mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)

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
		want    []*message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "successful round validation failed event",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"error_message": ["Invalid input", "Another error"]}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func() {
				expectedPayload := roundevents.RoundValidationFailedPayload{
					ErrorMessage: []string{"Invalid input", "Another error"},
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundValidationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundValidationFailedPayload) = expectedPayload
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(1)

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(1)
			},
		},
		{
			name: "failed to unmarshal payload",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`invalid payload`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(errors.New("unmarshal error")).Times(1)
			},
		},
		{
			name: "failed to update interaction response",
			fields: fields{
				Logger:       mockLogger,
				Helpers:      mockHelpers,
				RoundDiscord: mockRoundDiscord,
			},
			args: args{
				msg: message.NewMessage("1", []byte(`{"error_message": ["Invalid input", "Another error"]}`)),
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				expectedPayload := roundevents.RoundValidationFailedPayload{
					ErrorMessage: []string{"Invalid input", "Another error"},
				}
				mockHelpers.EXPECT().
					UnmarshalPayload(gomock.Any(), gomock.AssignableToTypeOf(&roundevents.RoundValidationFailedPayload{})).
					DoAndReturn(func(_ *message.Message, v any) error {
						*v.(*roundevents.RoundValidationFailedPayload) = expectedPayload
						return nil
					}).Times(1)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					Times(1)

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error")).Times(1)
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
			got, err := h.HandleRoundValidationFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundHandlers.HandleRoundValidationFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}
