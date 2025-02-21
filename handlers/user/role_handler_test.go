package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	discord_mocks "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleRoleUpdateCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := discord_mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	validPayload := discorduserevents.RoleUpdateCommandPayload{
		TargetUserID: "123456789",
	}
	payloadBytes, _ := json.Marshal(validPayload)

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful command handling",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("interaction_id", "int123")
				msg.Metadata.Set("interaction_token", "token123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any())
			},
			expectedError:  false,
			expectedResult: nil,
		},
		{
			name: "missing interaction metadata",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any())
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "interaction respond error",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("interaction_id", "int123")
				msg.Metadata.Set("interaction_token", "token123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(fmt.Errorf("interaction failed"))
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any())
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
			},
			expectedError:  true,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			h := &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}

			got, err := h.HandleRoleUpdateCommand(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleRoleUpdateCommand() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(got, tt.expectedResult) {
				t.Errorf("HandleRoleUpdateCommand() = %v, want %v", got, tt.expectedResult)
			}
		})
	}
}
func TestUserHandlers_HandleRoleUpdateButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := discord_mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	validPayload := discorduserevents.RoleUpdateButtonPressPayload{
		RequesterID:      "req123",
		TargetUserID:     "target123",
		InteractionID:    "role_button_Rattler",
		InteractionToken: "token123",
	}
	payloadBytes, _ := json.Marshal(validPayload)

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful button press handling",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(nil)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", nil) // ID and payload will be set by handler
					msg.Metadata.Set("interaction_token", "token123")
					return msg
				}(),
			},
		},
		{
			name: "interaction respond error",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", payloadBytes)
				msg.SetContext(context.Background())
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().InteractionRespond(gomock.Any(), gomock.Any()).Return(fmt.Errorf("interaction failed"))
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "invalid role button press",
			msg: func() *message.Message {
				invalidPayload := discorduserevents.RoleUpdateButtonPressPayload{
					RequesterID:      "req123",
					TargetUserID:     "target123",
					InteractionID:    "role_button_InvalidRole",
					InteractionToken: "token123",
				}
				invalidPayloadBytes, _ := json.Marshal(invalidPayload)
				msg := message.NewMessage("test-id", invalidPayloadBytes)
				msg.SetContext(context.Background())
				return msg
			}(),
			setupMocks: func() {
				// Expect the interaction response with `gomock.Any()` for flexible matching
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Any(),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).Return(fmt.Errorf("invalid role")) // Return error for invalid role

				// Expect error logging
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to acknowledge interaction",
						gomock.Any(),
					).AnyTimes()

				// Expect info logging
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError:  true, // Changed to true since we expect an error
			expectedResult: nil,
		},
		{
			name: "cancel button press",
			msg: func() *message.Message {
				cancelPayload := discorduserevents.RoleUpdateButtonPressPayload{
					RequesterID:      "req123",
					TargetUserID:     "target123",
					InteractionID:    "role_button_cancel",
					InteractionToken: "token123",
				}
				cancelPayloadBytes, _ := json.Marshal(cancelPayload)
				msg := message.NewMessage("test-id", cancelPayloadBytes)
				msg.SetContext(context.Background())
				return msg
			}(),
			setupMocks: func() {
				// Expect interaction response with the processing message
				mockSession.EXPECT().
					InteractionRespond(
						gomock.Any(),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).DoAndReturn(func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
					// Verify response type
					if resp.Type != discordgo.InteractionResponseUpdateMessage {
						return fmt.Errorf("expected UpdateMessage type, got %v", resp.Type)
					}
					expectedContent := "<@req123> has requested role 'cancel' for <@target123>. Request is being processed."
					if resp.Data == nil || resp.Data.Content != expectedContent {
						return fmt.Errorf("expected '%s', got '%s'", expectedContent, resp.Data.Content)
					}
					return nil
				})
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", nil)
					msg.Metadata.Set("interaction_token", "token123")
					return msg
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			h := &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}

			got, err := h.HandleRoleUpdateButtonPress(tt.msg)
			if (err != nil) != tt.expectedError {
				t.Errorf("HandleRoleUpdateButtonPress() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			// For successful cases, verify the returned message contains expected metadata
			if err == nil && got != nil {
				if len(got) != len(tt.expectedResult) {
					t.Errorf("Expected %d message(s), got %d", len(tt.expectedResult), len(got))
				} else if len(got) > 0 {
					if token := got[0].Metadata.Get("interaction_token"); token != validPayload.InteractionToken {
						t.Errorf("Expected interaction_token %s, got %s", validPayload.InteractionToken, token)
					}
				}
			}
		})
	}
}

func TestUserHandlers_HandleRoleUpdateResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := discord_mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	// Define common test data
	successPayload := userevents.UserRoleUpdateResultPayload{
		Error: "", // Only field actually used in handler
	}
	successPayloadBytes, err := json.Marshal(successPayload)
	if err != nil {
		t.Fatalf("Failed to marshal success payload: %v", err)
	}

	failurePayload := userevents.UserRoleUpdateResultPayload{
		Error: "permission denied",
	}
	failurePayloadBytes, err := json.Marshal(failurePayload)
	if err != nil {
		t.Fatalf("Failed to marshal failure payload: %v", err)
	}

	tests := []struct {
		name           string
		setupMessage   func() *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful role update",
			setupMessage: func() *message.Message {
				msg := message.NewMessage("test-id", successPayloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("topic", "role_update_result")
				msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
				return msg
			},
			setupMocks: func() {
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.WebhookEdit{}),
					).DoAndReturn(func(i *discordgo.Interaction, edit *discordgo.WebhookEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					if i.Token != "token123" {
						return nil, fmt.Errorf("expected token 'token123', got '%s'", i.Token)
					}
					expectedContent := "Role update completed"
					if *edit.Content != expectedContent {
						return nil, fmt.Errorf("expected content '%s', got '%s'", expectedContent, *edit.Content)
					}
					return &discordgo.Message{}, nil
				})
				mockLogger.EXPECT().Info(
					gomock.Any(),
					"Received role update result",
					attr.Topic("role_update_result"),
					gomock.Any(),
				)
			},
			expectedError:  false,
			expectedResult: nil,
		},
		{
			name: "failed role update",
			setupMessage: func() *message.Message {
				msg := message.NewMessage("test-id", failurePayloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("topic", userevents.UserRoleUpdateFailed)
				msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
				return msg
			},
			setupMocks: func() {
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.WebhookEdit{}),
					).DoAndReturn(func(i *discordgo.Interaction, edit *discordgo.WebhookEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
					if i.Token != "token123" {
						return nil, fmt.Errorf("expected token 'token123', got '%s'", i.Token)
					}
					expectedContent := "Failed to update role: permission denied"
					if *edit.Content != expectedContent {
						return nil, fmt.Errorf("expected content '%s', got '%s'", expectedContent, *edit.Content)
					}
					return &discordgo.Message{}, nil
				})
				mockLogger.EXPECT().Info(
					gomock.Any(),
					"Received role update result",
					attr.Topic(userevents.UserRoleUpdateFailed),
					gomock.Any(),
				)
			},
			expectedError:  false,
			expectedResult: nil,
		},
		{
			name: "missing interaction token",
			setupMessage: func() *message.Message {
				msg := message.NewMessage("test-id", successPayloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("topic", "role_update_result")
				msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
				return msg
			},
			setupMocks: func() {
				mockLogger.EXPECT().Info(
					gomock.Any(),
					"Received role update result",
					attr.Topic("role_update_result"),
					gomock.Any(),
				)
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"interaction_token missing from metadata",
					gomock.Any(),
				)
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "interaction response edit error",
			setupMessage: func() *message.Message {
				msg := message.NewMessage("test-id", successPayloadBytes)
				msg.SetContext(context.Background())
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("topic", "role_update_result")
				msg.Metadata.Set("handler_name", "HandleRoleUpdateResult")
				return msg
			},
			setupMocks: func() {
				mockSession.EXPECT().
					InteractionResponseEdit(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.WebhookEdit{}),
					).Return(nil, fmt.Errorf("edit failed"))
				mockLogger.EXPECT().Info(
					gomock.Any(),
					"Received role update result",
					attr.Topic("role_update_result"),
					gomock.Any(),
				)
				mockLogger.EXPECT().Error(
					gomock.Any(),
					"Failed to edit interaction response",
					gomock.Any(),
				)
			},
			expectedError:  true,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			h := &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}

			got, err := h.HandleRoleUpdateResult(tt.setupMessage())

			if (err != nil) != tt.expectedError {
				t.Errorf("HandleRoleUpdateResult() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(got, tt.expectedResult) {
				t.Errorf("HandleRoleUpdateResult() = %v, want %v", got, tt.expectedResult)
			}
		})
	}
}
