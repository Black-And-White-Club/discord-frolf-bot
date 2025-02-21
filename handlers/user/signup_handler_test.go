package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleSignupSubmission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{Discord: config.DiscordConfig{DiscordAppID: "test_app_id"}}
	mockEventUtil := utils.NewEventUtil()

	// Define common interaction responses
	successInteraction := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Your signup has been submitted!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	invalidTagInteraction := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<@12345> Invalid tag number format.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	errorInteraction := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "An error occurred processing your signup. Please try again.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful signup - with tag",
			msg: func() *message.Message {
				payload := []byte(`{
							"type": 5,
							"data": {
									"custom_id": "signup_modal",
									"components": [
											{
													"type": 1,
													"components": [
															{
																	"type": 4,
																	"custom_id": "tag_number",
																	"value": "12345"
															}
													]
											}
									]
							},
							"member": {"user": {"id": "12345"}},
							"id": "interaction123",
							"token": "token123"
					}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("user_id", "123456789")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(gomock.Any(), "Processing signup submission", gomock.Any())

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Eq(successInteraction)).
					Return(nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
									"domain": "discord_user",
									"event_name": "discord.user.SignupFormSubmitted",
									"user_id": "12345",
									"interaction_id": "interaction123",
									"interaction_token": "token123",
									"tag_number": 12345,
									"timestamp": "0001-01-01T00:00:00Z"
							}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("domain", "discord_user")
					msg.Metadata.Set("event_name", "discord.user.SignupFormSubmitted")
					msg.Metadata.Set("handler_name", "HandleSignupSubmission")
					return msg
				}(),
			},
		},
		{
			name: "successful signup - no tag",
			msg: func() *message.Message {
				payload := []byte(`{
									"type": 5,
									"data": {
											"custom_id": "signup_modal",
											"components": []
									},
									"member": {"user": {"id": "12345"}},
									"id": "interaction123",
									"token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("user_id", "123456789")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(gomock.Any(), "Processing signup submission", gomock.Any())

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Eq(successInteraction)).
					Return(nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"domain": "discord_user",
											"event_name": "discord.user.SignupFormSubmitted",
											"user_id": "12345",
											"interaction_id": "interaction123",
											"interaction_token": "token123",
											"tag_number": null,
											"timestamp": "0001-01-01T00:00:00Z"
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "123456789")
					msg.Metadata.Set("domain", "discord_user")
					msg.Metadata.Set("event_name", "discord.user.SignupFormSubmitted")
					msg.Metadata.Set("handler_name", "HandleSignupSubmission")
					return msg
				}(),
			},
		},
		{
			name: "invalid tag format",
			msg: func() *message.Message {
				payload := []byte(`{
									"type": 5,
									"data": {
											"custom_id": "signup_modal",
											"components": [
													{
															"type": 1,
															"components": [
																	{
																			"type": 4,
																			"custom_id": "tag_number",
																			"value": "invalid"
																	}
															]
													}
											]
									},
									"member": {"user": {"id": "12345"}},
									"id": "interaction123",
									"token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("user_id", "123456789")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(gomock.Any(), "Processing signup submission", gomock.Any())

				mockLogger.EXPECT().
					Warn(gomock.Any(), "Invalid tag number", gomock.Any(), gomock.Any())

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Eq(invalidTagInteraction)).
					Return(fmt.Errorf("invalid tag"))
			},
			expectedError:  true,
			expectedResult: nil,
		},
		{
			name: "incorrect modal ID",
			msg: func() *message.Message {
				payload := []byte(`{
									"type": 5,
									"data": {
											"custom_id": "wrong_modal",
											"components": []
									},
									"member": {"user": {"id": "12345"}},
									"id": "interaction123",
									"token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("user_id", "123456789")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Info(gomock.Any(), "Processing signup submission", gomock.Any())

				mockLogger.EXPECT().
					Error(gomock.Any(), "Incorrect modal ID", gomock.Any())

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Eq(errorInteraction)).
					Return(fmt.Errorf("wrong modal"))
			},
			expectedError:  true,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signupCache = sync.Map{} // Clear cache before each test
			tt.setupMocks()

			h := &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}

			got, err := h.HandleSignupSubmission(tt.msg)

			// Verify error expectations
			if tt.expectedError && err == nil {
				t.Error("expected an error but got nil")
				return
			}
			if !tt.expectedError && err != nil {
				t.Errorf("expected no error but got: %v", err)
				return
			}

			// Verify result expectations
			if tt.expectedResult == nil {
				if got != nil {
					t.Error("expected nil result but got messages")
				}
				return
			}

			if len(got) != len(tt.expectedResult) {
				t.Errorf("expected %d messages, got %d", len(tt.expectedResult), len(got))
				return
			}

			for i, expectedMsg := range tt.expectedResult {
				var gotPayload, expectedPayload map[string]interface{}
				if err := json.Unmarshal(got[i].Payload, &gotPayload); err != nil {
					t.Fatalf("failed to unmarshal got payload: %v", err)
				}
				if err := json.Unmarshal(expectedMsg.Payload, &expectedPayload); err != nil {
					t.Fatalf("failed to unmarshal expected payload: %v", err)
				}

				if !reflect.DeepEqual(gotPayload, expectedPayload) {
					t.Errorf("message %d payload mismatch:\ngot:  %v\nwant: %v", i, gotPayload, expectedPayload)
				}

				if !reflect.DeepEqual(got[i].Metadata, expectedMsg.Metadata) {
					t.Errorf("message %d metadata mismatch:\ngot:  %v\nwant: %v", i, got[i].Metadata, expectedMsg.Metadata)
				}
			}
		})
	}
}

func TestUserHandlers_HandleSignupFormSubmitted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{Discord: config.DiscordConfig{DiscordAppID: "test_app_id"}}
	mockEventUtil := utils.NewEventUtil()

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful signup - no tag",
			msg: func() *message.Message {
				payload := []byte(`{
									"user_id": "12345",
									"interaction_id": "interaction123",
									"interaction_token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("user_id", "123456789")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).Return(nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"discord_id": "12345"
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "123456789")
					return msg
				}(),
			},
		},
		{
			name: "successful signup - with tag",
			msg: func() *message.Message {
				payload := []byte(`{
									"user_id": "12345",
									"interaction_id": "interaction123",
									"interaction_token": "token123",
									"tag_number": 54321
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("user_id", "123456789")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					InteractionRespond(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).Return(nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"discord_id": "12345",
											"tag_number": 54321
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "123456789")
					return msg
				}(),
			},
		},
		{
			name: "duplicate signup attempt",
			msg: func() *message.Message {
				payload := []byte(`{
									"user_id": "12345",
									"interaction_id": "interaction123",
									"interaction_token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("user_id", "123456789")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				// Simulate existing signup
				signupCache.Store("12345", struct{}{})

				mockLogger.EXPECT().
					Warn(gomock.Any(), "Duplicate signup attempt detected", gomock.Any())
				mockSession.EXPECT().
					InteractionRespond(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).Return(nil)
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "123456789",
											"result": "failure",
											"reason": "Duplicate signup attempt",
											"event_name": "discord.signup.failed"
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "123456789")
					return msg
				}(),
			},
		},
		{
			name: "interaction respond error",
			msg: func() *message.Message {
				payload := []byte(`{
									"user_id": "12345",
									"interaction_id": "interaction123",
									"interaction_token": "token123"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("user_id", "12345")
				msg.Metadata.Set("interaction_id", "interaction123")
				msg.Metadata.Set("interaction_token", "token123")
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Error(gomock.Any(), "Failed to send interaction response", gomock.Any(), gomock.Any())
				mockSession.EXPECT().
					InteractionRespond(
						gomock.AssignableToTypeOf(&discordgo.Interaction{}),
						gomock.AssignableToTypeOf(&discordgo.InteractionResponse{}),
					).Return(fmt.Errorf("simulated error"))
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "12345",
											"result": "failure",
											"reason": "internal error",
											"event_name": "discord.signup.failed"
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					msg.Metadata.Set("interaction_id", "interaction123")
					msg.Metadata.Set("interaction_token", "token123")
					msg.Metadata.Set("user_id", "12345")
					return msg
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signupCache = sync.Map{} // Clear cache before each test
			tt.setupMocks()

			h := &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			}

			got, err := h.HandleSignupFormSubmitted(tt.msg)

			if (err != nil) != tt.expectedError {
				t.Errorf("HandleSignupFormSubmitted() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if len(got) != len(tt.expectedResult) {
					t.Errorf("HandleSignupFormSubmitted() got %v messages, want %v", len(got), len(tt.expectedResult))
					return
				}

				for i := range got {
					var gotPayload, wantPayload interface{}
					if err := json.Unmarshal(got[i].Payload, &gotPayload); err != nil {
						t.Fatalf("Failed to unmarshal got payload: %v", err)
					}
					if err := json.Unmarshal(tt.expectedResult[i].Payload, &wantPayload); err != nil {
						t.Fatalf("Failed to unmarshal want payload: %v", err)
					}

					if !reflect.DeepEqual(gotPayload, wantPayload) {
						t.Errorf("HandleSignupFormSubmitted() payload mismatch:\ngot  = %v\nwant = %v",
							gotPayload, wantPayload)
					}

					if !reflect.DeepEqual(got[i].Metadata, tt.expectedResult[i].Metadata) {
						t.Errorf("HandleSignupFormSubmitted() metadata mismatch:\ngot  = %v\nwant = %v",
							got[i].Metadata, tt.expectedResult[i].Metadata)
					}
				}
			}
		})
	}
}
