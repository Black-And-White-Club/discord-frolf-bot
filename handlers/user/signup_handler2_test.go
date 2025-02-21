package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful user created",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(&discordgo.Channel{ID: "channel123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend(
						"channel123",
						"Signup complete! You now have access to the members-only channels.",
					).Return(&discordgo.Message{}, nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup complete! You now have access to the members-only channels."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "failed to create DM channel",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(nil, fmt.Errorf("channel creation error"))
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to create DM channel",
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup complete! You now have access to the members-only channels."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "failed to send DM",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(&discordgo.Channel{ID: "channel123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend(
						"channel123",
						"Signup complete! You now have access to the members-only channels.",
					).Return(nil, fmt.Errorf("message send error"))
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to send DM",
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup complete! You now have access to the members-only channels."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "invalid payload",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", []byte(`invalid-json`))
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to unmarshal UserCreatedPayload",
						gomock.Any(),
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

			got, err := h.HandleUserCreated(tt.msg)

			if (err != nil) != tt.expectedError {
				t.Errorf("HandleUserCreated() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if len(got) != len(tt.expectedResult) {
					t.Errorf("HandleUserCreated() got %v messages, want %v", len(got), len(tt.expectedResult))
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
						t.Errorf("HandleUserCreated() payload mismatch:\ngot  = %v\nwant = %v",
							gotPayload, wantPayload)
					}

					if !reflect.DeepEqual(got[i].Metadata, tt.expectedResult[i].Metadata) {
						t.Errorf("HandleUserCreated() metadata mismatch:\ngot  = %v\nwant = %v",
							got[i].Metadata, tt.expectedResult[i].Metadata)
					}
				}
			}
		})
	}
}

func TestUserHandlers_HandleUserCreationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	tests := []struct {
		name           string
		msg            *message.Message
		setupMocks     func()
		expectedError  bool
		expectedResult []*message.Message
	}{
		{
			name: "successful failure handling",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345",
									"reason": "some reason"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(&discordgo.Channel{ID: "channel123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend(
						"channel123",
						"Signup failed: some reason. Please try again, or contact an administrator.",
					).Return(&discordgo.Message{}, nil)
			},
			expectedError: false,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup failed: some reason. Please try again, or contact an administrator."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "failed to create DM channel",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345",
									"reason": "some reason"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(nil, fmt.Errorf("channel creation error"))
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to create DM channel",
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup failed: some reason. Please try again, or contact an administrator."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "failed to send DM",
			msg: func() *message.Message {
				payload := []byte(`{
									"discord_id": "12345",
									"reason": "some reason"
							}`)
				msg := message.NewMessage("test-id", payload)
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockSession.EXPECT().
					UserChannelCreate("12345").
					Return(&discordgo.Channel{ID: "channel123"}, nil)
				mockSession.EXPECT().
					ChannelMessageSend(
						"channel123",
						"Signup failed: some reason. Please try again, or contact an administrator.",
					).Return(nil, fmt.Errorf("message send error"))
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to send DM",
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					)
			},
			expectedError: true,
			expectedResult: []*message.Message{
				func() *message.Message {
					msg := message.NewMessage("", []byte(`{
											"user_id": "",
											"message": "Signup failed: some reason. Please try again, or contact an administrator."
									}`))
					msg.Metadata.Set("correlation_id", "correlation123")
					return msg
				}(),
			},
		},
		{
			name: "invalid payload",
			msg: func() *message.Message {
				msg := message.NewMessage("test-id", []byte(`invalid-json`))
				msg.SetContext(context.Background())
				msg.Metadata.Set("correlation_id", "correlation123")
				return msg
			}(),
			setupMocks: func() {
				mockLogger.EXPECT().
					Error(
						gomock.Any(),
						"Failed to unmarshal UserCreationFailedPayload",
						gomock.Any(),
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

			got, err := h.HandleUserCreationFailed(tt.msg)

			if (err != nil) != tt.expectedError {
				t.Errorf("HandleUserCreationFailed() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if tt.expectedResult != nil {
				if len(got) != len(tt.expectedResult) {
					t.Errorf("HandleUserCreationFailed() got %v messages, want %v", len(got), len(tt.expectedResult))
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
						t.Errorf("HandleUserCreationFailed() payload mismatch:\ngot  = %v\nwant = %v",
							gotPayload, wantPayload)
					}

					if !reflect.DeepEqual(got[i].Metadata, tt.expectedResult[i].Metadata) {
						t.Errorf("HandleUserCreationFailed() metadata mismatch:\ngot  = %v\nwant = %v",
							got[i].Metadata, tt.expectedResult[i].Metadata)
					}
				}
			}
		})
	}
}
