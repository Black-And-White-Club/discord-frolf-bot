package userhandlers

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	loggermocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

// Helper function to create a mock message.
func createMockMessageWithMetadata(userID, correlationID string, payload []byte) *message.Message {
	msg := message.NewMessage("test-msg-id", payload)
	msg.Metadata.Set("user_id", userID)
	msg.Metadata.Set("correlation_id", correlationID)
	return msg
}

// Helper function to assert metadata.
func assertMetadata(t *testing.T, got *message.Message, want *message.Message, exclude ...string) {
	t.Helper()
	excludeMap := make(map[string]bool)
	for _, key := range exclude {
		excludeMap[key] = true
	}

	for key, wantVal := range want.Metadata {
		if _, excluded := excludeMap[key]; excluded {
			continue
		}
		if gotVal := got.Metadata.Get(key); gotVal != wantVal {
			t.Errorf("Metadata mismatch for key '%s': got '%s', want '%s'", key, gotVal, wantVal)
		}
	}
}

// Helper function to assert payload, handling timestamps separately.
func assertPayload(t *testing.T, got *message.Message, want *message.Message) {
	t.Helper()

	var gotPayload map[string]interface{}
	var wantPayload map[string]interface{}

	if err := json.Unmarshal(got.Payload, &gotPayload); err != nil {
		t.Fatalf("failed to unmarshal got payload: %v", err)
	}
	if err := json.Unmarshal(want.Payload, &wantPayload); err != nil {
		t.Fatalf("failed to unmarshal want payload: %v", err)
	}

	// Check for the *existence* and *type* of the timestamp field in gotPayload.
	gotTimestamp, ok := gotPayload["timestamp"]
	if !ok {
		t.Fatal("got payload missing 'timestamp' field")
	}
	if _, ok := gotTimestamp.(string); !ok {
		t.Fatalf("got payload 'timestamp' field is not a string: %T", gotTimestamp)
	}

	// Check for the *existence* of the timestamp field in wantPayload.
	_, ok = wantPayload["timestamp"]
	if !ok {
		t.Fatal("want payload missing 'timestamp' field")
	}

	// Remove timestamps before comparing the rest of the payload.
	delete(gotPayload, "timestamp")
	delete(wantPayload, "timestamp")

	if !reflect.DeepEqual(gotPayload, wantPayload) {
		// If they're still not equal, *then* print as JSON for better readability.
		gotJSON, _ := json.MarshalIndent(gotPayload, "", "  ")   // Pretty print
		wantJSON, _ := json.MarshalIndent(wantPayload, "", "  ") // Pretty print
		t.Errorf("Payload mismatch:\nGot:\n%s\nWant:\n%s", string(gotJSON), string(wantJSON))
	}
}

// Helper function to create the expected DMSentPayload.
func expectedDMSentPayload(userID, channelID string) []byte {
	expectedPayload := discorduserevents.DMSentPayload{
		UserID:    userID,
		ChannelID: channelID,
		CommonMetadata: events.CommonMetadata{
			Domain:    userDomain,
			EventName: discorduserevents.DMSent,
		},
	}
	payloadBytes, _ := json.Marshal(expectedPayload) // Ignoring error, as it's a controlled struct.
	return payloadBytes
}

// Helper function to create the expected DMErrorPayload.
func expectedDMErrorPayload(userID, errorDetail, eventName string) []byte {
	expectedPayload := discorduserevents.DMErrorPayload{
		UserID:      userID,
		ErrorDetail: errorDetail,
		CommonMetadata: events.CommonMetadata{
			Domain:    userDomain,
			EventName: eventName,
		},
	}
	payloadBytes, _ := json.Marshal(expectedPayload)
	return payloadBytes
}

// Helper function to create expected InteractionRespondedPayload
func expectedInteractionRespondedPayload(userID, status, reason, eventName, interactionID string) []byte {
	expectedPayload := discorduserevents.InteractionRespondedPayload{
		UserID:        userID,
		Status:        status,
		ErrorDetail:   reason,
		InteractionID: interactionID,
		CommonMetadata: events.CommonMetadata{
			Domain:    userDomain,
			EventName: eventName,
		},
	}
	payloadBytes, _ := json.Marshal(expectedPayload)
	return payloadBytes
}

// anyError is a custom matcher that matches any non-nil error.
type anyError struct{}

func (anyError) Matches(x interface{}) bool {
	_, ok := x.(error)
	return ok
}

func (anyError) String() string {
	return "is any error"
}

// isAnyError returns a matcher that matches any non-nil error.
func isAnyError() gomock.Matcher {
	return anyError{}
}

func TestUserHandlers_HandleSendUserDM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockSession(ctrl)
	mockLogger := loggermocks.NewMockLogger(ctrl)
	mockEventUtil := utils.NewEventUtil()

	type fields struct {
		Logger    observability.Logger
		Session   discord.Session
		Config    *config.Config
		EventUtil utils.EventUtil
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       []*message.Message
		setupMocks func()
		wantErr    bool
		errMsg     string // Keep for unmarshal errors
	}{
		{
			name: "Successful DM Send",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "message": "Hello!"}`)),
			},
			setupMocks: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").Return(&discordgo.Channel{ID: "channel123"}, nil).Times(1)
				mockSession.EXPECT().ChannelMessageSend("channel123", "Hello!").Return(&discordgo.Message{}, nil).Times(1)
				mockLogger.EXPECT().Info(gomock.Any(), "Sending DM", gomock.Any(), gomock.Any()).Times(1)
				mockLogger.EXPECT().Info(gomock.Any(), "DM sent successfully", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", expectedDMSentPayload("123456789", "channel123"))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMSent)
					msg.Metadata.Set("domain", userDomain)
					return msg
				}(),
			},
			wantErr: false,
		},
		{
			name: "Failed to Create DM Channel",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "message": "Hello!"}`)),
			},
			setupMocks: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").Return(nil, errors.New("channel creation error")).Times(1)
				mockLogger.EXPECT().Info(gomock.Any(), "Sending DM", gomock.Any(), gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(gomock.Any(), "DM operation failed", gomock.Any(), attr.UserID("123456789"), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", expectedDMErrorPayload("123456789", "failed to create DM channel: channel creation error", discorduserevents.DMCreateError))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMCreateError)
					msg.Metadata.Set("domain", userDomain)
					return msg
				}(),
			},
			wantErr: true,
		},
		{
			name: "Failed to Send DM",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "message": "Hello!"}`)),
			},
			setupMocks: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").Return(&discordgo.Channel{ID: "channel123"}, nil).Times(1)
				mockSession.EXPECT().ChannelMessageSend("channel123", "Hello!").Return(nil, errors.New("message send error")).Times(1)
				mockLogger.EXPECT().Info(gomock.Any(), "Sending DM", gomock.Any(), gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(gomock.Any(), "DM operation failed", gomock.Any(), attr.UserID("123456789"), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", expectedDMErrorPayload("123456789", "failed to send DM: message send error", discorduserevents.DMSendError))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMSendError)
					msg.Metadata.Set("domain", userDomain)
					return msg
				}(),
			},
			wantErr: true,
		},
		{
			name: "Invalid Payload",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`invalid-json`)),
			},
			setupMocks: func() {
				mockSession.EXPECT().UserChannelCreate(gomock.Any()).Times(0)
				mockSession.EXPECT().ChannelMessageSend(gomock.Any(), gomock.Any()).Times(0)
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"Failed to unmarshal payload",
					gomock.Any(),
					gomock.Any(),
				).Times(1)
			},
			want:    nil,
			wantErr: true,
			errMsg:  "failed to unmarshal payload",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &UserHandlers{
				Logger:    tt.fields.Logger,
				Session:   tt.fields.Session,
				Config:    tt.fields.Config,
				EventUtil: tt.fields.EventUtil,
			}
			tt.setupMocks()
			got, err := h.HandleSendUserDM(tt.args.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleSendUserDM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Keep this check for unmarshal errors, where a *specific* error is expected.
			if tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("UserHandlers.HandleSendUserDM() error = %v, wantErrMsg %v", err.Error(), tt.errMsg)
			}

			if tt.wantErr == false && tt.want != nil {
				if len(got) != len(tt.want) {
					t.Errorf("UserHandlers.HandleSendUserDM() got %v messages, want %v", len(got), len(tt.want))
					return
				}
				for i := range got {
					assertMetadata(t, got[i], tt.want[i], "handler_name") // Exclude handler_name
					assertPayload(t, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUserHandlers_HandleDMSent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockSession(ctrl)
	mockLogger := loggermocks.NewMockLogger(ctrl)
	mockEventUtil := utils.NewEventUtil()

	type fields struct {
		Logger    observability.Logger
		Session   discord.Session
		Config    *config.Config
		EventUtil utils.EventUtil
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       []*message.Message
		setupMocks func()
		wantErr    bool
		errMsg     string
	}{
		{
			name: "Successful DM Sent Confirmation",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				//Payload doesn't matter, as long as valid
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "channel_id": "channel123"}`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Info(gomock.Any(), "DM sent successfully (confirmation received)", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", expectedInteractionRespondedPayload("123456789", interactionSuccess, "", discorduserevents.DMSent, ""))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMSent) //We know this from the interactionResponded helper.
					msg.Metadata.Set("domain", userDomain)                   // We know this from the interactionResponded helper.
					msg.Metadata.Set("topic", interactionRespondedTopic)
					return msg
				}(),
			},
			wantErr: false,
		},
		{
			name: "Invalid Payload",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`invalid-json`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"Failed to unmarshal payload",
					gomock.Any(),
					gomock.Any(),
				).Times(1)
			},
			wantErr: true,
			errMsg:  "failed to unmarshal payload",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &UserHandlers{
				Logger:    tt.fields.Logger,
				Session:   tt.fields.Session,
				Config:    tt.fields.Config,
				EventUtil: tt.fields.EventUtil,
			}
			if tt.setupMocks != nil {
				tt.setupMocks()
			}
			got, err := h.HandleDMSent(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleDMSent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("UserHandlers.HandleDMSent() error = %v, wantErrMsg %v", err, tt.errMsg)

			}
			if tt.wantErr == false && tt.want != nil {
				if len(got) != len(tt.want) {
					t.Errorf("UserHandlers.HandleDMSent() got %v messages, want %v", len(got), len(tt.want))
					return
				}
				for i := range got {
					assertMetadata(t, got[i], tt.want[i], "handler_name") // Exclude handler_name
					assertPayload(t, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUserHandlers_HandleDMCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockSession(ctrl)
	mockLogger := loggermocks.NewMockLogger(ctrl)
	mockEventUtil := utils.NewEventUtil()

	type fields struct {
		Logger    observability.Logger
		Session   discord.Session
		Config    *config.Config
		EventUtil utils.EventUtil
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       []*message.Message
		setupMocks func()
		wantErr    bool
		errMsg     string
	}{
		{
			name: "Successful DM Create Error Handling",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "error_detail": "Some error", "event_name":"discord.user.dmcreateerror", "domain":"user"}`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Error(gomock.Any(), "DM operation failed", gomock.Any(), attr.UserID("123456789"), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", expectedInteractionRespondedPayload("123456789", interactionFailure, "Some error", discorduserevents.DMCreateError, ""))
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMCreateError)
					msg.Metadata.Set("domain", userDomain)
					msg.Metadata.Set("topic", interactionRespondedTopic)
					msg.Metadata.Set("correlation_id", "correlation123")

					return msg
				}(),
			},
			wantErr: false,
		},
		{
			name: "Invalid payload",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`invalid-json`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"Failed to unmarshal payload",
					gomock.Any(),
					gomock.Any(),
				).Times(1)
			},

			wantErr: true,
			errMsg:  "failed to unmarshal payload",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &UserHandlers{
				Logger:    tt.fields.Logger,
				Session:   tt.fields.Session,
				Config:    tt.fields.Config,
				EventUtil: tt.fields.EventUtil,
			}
			if tt.setupMocks != nil {
				tt.setupMocks()
			}
			got, err := h.HandleDMCreateError(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleDMCreateError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errMsg != "" && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("UserHandlers.HandleDMCreateError() error = %v, wantErrMsg %v", err, tt.errMsg)
			}
			if tt.wantErr == false && tt.want != nil {
				if len(got) != len(tt.want) {
					t.Errorf("UserHandlers.HandleDMCreateError() got %v messages, want %v", len(got), len(tt.want))
					return
				}
				for i := range got {
					assertMetadata(t, got[i], tt.want[i], "handler_name") // Exclude handler_name
					assertPayload(t, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUserHandlers_HandleDMSendError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := mocks.NewMockSession(ctrl)
	mockLogger := loggermocks.NewMockLogger(ctrl)
	mockEventUtil := utils.NewEventUtil()

	type fields struct {
		Logger    observability.Logger
		Session   discord.Session
		Config    *config.Config
		EventUtil utils.EventUtil
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       []*message.Message
		setupMocks func()
		wantErr    bool
		errMsg     string
	}{
		{
			name: "Successful DM Send Error Handling",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`{"user_id": "123456789", "error_detail": "Some send error"}`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Error(gomock.Any(), "DM operation failed", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			},
			want: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
						"domain": "discord_user",
						"event_name": "discord.user.dmsenderror",
						"user_id": "123456789",
						"status": "failure",
						"error_detail": "Some send error"
					}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("event_name", discorduserevents.DMSendError)
					msg.Metadata.Set("domain", userDomain)
					msg.Metadata.Set("handler_name", "HandleDMSendError")
					msg.Metadata.Set("topic", interactionRespondedTopic)

					return msg
				}(),
			},
			wantErr: false,
		},
		{
			name: "Invalid payload",
			fields: fields{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    &config.Config{},
				EventUtil: mockEventUtil,
			},
			args: args{
				msg: createMockMessageWithMetadata("123456789", "correlation123", []byte(`invalid-json`)),
			},
			setupMocks: func() {
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"Failed to unmarshal payload",
					gomock.Any(),
					gomock.Any(),
				).Times(1)
			},
			wantErr: true,
			errMsg:  "failed to unmarshal payload: invalid character 'i' looking for beginning of value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &UserHandlers{
				Logger:    tt.fields.Logger,
				Session:   tt.fields.Session,
				Config:    tt.fields.Config,
				EventUtil: tt.fields.EventUtil,
			}
			if tt.setupMocks != nil {
				tt.setupMocks()
			}
			got, err := h.HandleDMSendError(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleDMSendError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
				t.Errorf("UserHandlers.HandleDMSendError() error = %v, wantErrMsg %v", err, tt.errMsg)
			}

			if tt.wantErr == false && tt.want != nil {
				if len(got) != len(tt.want) {
					t.Errorf("UserHandlers.HandleDMSendError() got %v messages, want %v", len(got), len(tt.want))
					return
				}
				for i := range got {
					// Compare metadata explicitly
					expectedMetadata := map[string]string{
						"correlation_id": "correlation123",
						"user_id":        "123456789",
						"event_name":     discorduserevents.DMSendError,
						"domain":         userDomain,
						"handler_name":   "HandleDMSendError",
						"topic":          interactionRespondedTopic,
					}
					for k, v := range expectedMetadata {
						if got[i].Metadata.Get(k) != v {
							t.Errorf("Metadata mismatch for key %s: got %v, want %v", k, got[i].Metadata.Get(k), v)
						}
					}

					// Compare payload while ignoring timestamp
					expectedPayloadJSON := `{
						"domain": "discord_user",
						"event_name": "discord.user.dmsenderror",
						"user_id": "123456789",
						"status": "failure",
						"error_detail": "Some send error"
					}`

					var gotPayload map[string]interface{}
					var expectedPayload map[string]interface{}

					json.Unmarshal(got[i].Payload, &gotPayload)
					json.Unmarshal([]byte(expectedPayloadJSON), &expectedPayload)

					// Remove dynamic fields before comparison
					delete(gotPayload, "timestamp")
					delete(gotPayload, "interaction_id")

					if !reflect.DeepEqual(gotPayload, expectedPayload) {
						t.Errorf("Payload mismatch: got %v, want %v", gotPayload, expectedPayload)
					}
				}
			}
		})
	}
}
