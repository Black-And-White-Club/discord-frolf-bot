package createround

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	utilsmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	logmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_createRoundManager_SendCreateRoundModal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := logmocks.NewMockLogger(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)

	// Sample interaction with member and user
	testInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID: "interaction-id",
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
		},
	}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "successful modal response",
			setup: func() {
				// Expect logging
				mockLogger.EXPECT().
					Info(gomock.Any(), "Sending create round modal", gomock.Any()).
					Times(1)

				// Expect modal to be sent successfully
				mockSession.EXPECT().
					InteractionRespond(gomock.Eq(testInteraction.Interaction), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {

						// Validate the response
						if r.Type != discordgo.InteractionResponseModal {
							t.Errorf("Expected InteractionResponseModal, got %v", r.Type)
						}
						if r.Data.Title != "Create Round" {
							t.Errorf("Expected title 'Create Round', got %v", r.Data.Title)
						}
						if r.Data.CustomID != "create_round_modal" {
							t.Errorf("Expected CustomID 'create_round_modal', got %v", r.Data.CustomID)
						}
						if len(r.Data.Components) != 5 {
							t.Errorf("Expected 5 components, got %v", len(r.Data.Components))
						}

						// Validate each component
						components := r.Data.Components

						// First component - Title
						firstRow, ok := components[0].(discordgo.ActionsRow)
						if !ok {
							t.Error("First component is not an ActionsRow")
						} else {
							textInput, ok := firstRow.Components[0].(discordgo.TextInput)
							if !ok {
								t.Error("First row component is not a TextInput")
							} else {
								if textInput.CustomID != "title" {
									t.Errorf("Expected CustomID 'title', got %v", textInput.CustomID)
								}
							}
						}

						// Second component - Description
						secondRow, ok := components[1].(discordgo.ActionsRow)
						if !ok {
							t.Error("Second component is not an ActionsRow")
						} else {
							textInput, ok := secondRow.Components[0].(discordgo.TextInput)
							if !ok {
								t.Error("Second row component is not a TextInput")
							} else {
								if textInput.CustomID != "description" {
									t.Errorf("Expected CustomID 'description', got %v", textInput.CustomID)
								}
							}
						}

						return nil
					})
			},
			wantErr: false,
		},
		{
			name: "error sending modal",
			setup: func() {
				// Expect logging
				mockLogger.EXPECT().
					Info(gomock.Any(), "Sending create round modal", gomock.Any()).
					Times(1)

				// Expect error response
				mockSession.EXPECT().
					InteractionRespond(gomock.Eq(testInteraction.Interaction), gomock.Any()).
					Return(errors.New("failed to send modal"))

				// Expect error to be logged
				mockLogger.EXPECT().
					Error(gomock.Any(), "Failed to send create round modal", gomock.Any(), gomock.Any()).
					Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test expectations
			if tt.setup != nil {
				tt.setup()
			}

			// Create the manager with mocked dependencies
			crm := &createRoundManager{
				session:          mockSession,
				logger:           mockLogger,
				interactionStore: mockInteractionStore,
			}

			// Call the function under test
			err := crm.SendCreateRoundModal(context.Background(), testInteraction)

			// Verify the result
			if (err != nil) != tt.wantErr {
				t.Errorf("createRoundManager.SendCreateRoundModal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_createRoundManager_HandleCreateRoundModalSubmit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockLogger := logmocks.NewMockLogger(ctrl)
	mockHelper := utilsmocks.NewMockHelpers(ctrl)

	// Helper function to create a test interaction with modal submit data
	createTestInteraction := func(title, description, startTime, timezone, location string) *discordgo.InteractionCreate {
		return &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				ID:    "interaction-id",
				Token: "interaction-token",
				Member: &discordgo.Member{
					User: &discordgo.User{
						ID: "user-123",
					},
				},
				Type: discordgo.InteractionModalSubmit,
				Data: discordgo.ModalSubmitInteractionData{
					CustomID: "create_round_modal",
					Components: []discordgo.MessageComponent{
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: "title",
									Value:    title,
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: "description",
									Value:    description,
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: "start_time",
									Value:    startTime,
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: "timezone",
									Value:    timezone,
								},
							},
						},
						&discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								&discordgo.TextInput{
									CustomID: "location",
									Value:    location,
								},
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name        string
		interaction *discordgo.InteractionCreate
		setup       func()
	}{
		{
			name:        "successful submission",
			interaction: createTestInteraction("Test Round", "Fun round description", "2025-04-01 14:00", "America/Chicago", "Disc Golf Park"),
			setup: func() {
				// Expect interaction store to be called
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Expect event creation and publication
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundCreateModalSubmit), gomock.Any()).
					Return(nil)

				// Expect final interaction response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if r.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", r.Type)
						}
						if !strings.Contains(r.Data.Content, "Round creation request received. Please wait for confirmation.") {
							t.Errorf("Expected success message, got %v", r.Data.Content)
						}
						return nil
					})
			},
		},
		{
			name:        "missing required fields",
			interaction: createTestInteraction("", "", "", "", ""),
			setup: func() {
				// Expect validation error response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if r.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %v", r.Type)
						}
						if !strings.Contains(r.Data.Content, "Round creation failed") {
							t.Errorf("Expected validation error message, got %v", r.Data.Content)
						}
						if !strings.Contains(r.Data.Content, "Title is required.") {
							t.Errorf("Expected 'Title is required.' in error message, got %v", r.Data.Content)
						}
						if !strings.Contains(r.Data.Content, "Start Time is required.") {
							t.Errorf("Expected 'Start Time is required.' in error message, got %v", r.Data.Content)
						}
						return nil
					})
			},
		},
		{
			name:        "default timezone",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "", "Location"),
			setup: func() {
				// Expect interaction store to be called
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Expect event creation and publication
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundCreateModalSubmit), gomock.Any()).
					DoAndReturn(func(topic string, msg *message.Message) error {
						// Extract payload to verify timezone
						var payload roundevents.CreateRoundRequestedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Errorf("Failed to unmarshal payload: %v", err)
						}

						if payload.Timezone != "America/Chicago" {
							t.Errorf("Expected default timezone 'America/Chicago', got %v", payload.Timezone)
						}
						return nil
					})

				// Expect successful interaction response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:        "field too long",
			interaction: createTestInteraction(strings.Repeat("A", 101), "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func() {
				// Expect validation error response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if !strings.Contains(r.Data.Content, "Title must be less than 100 characters.") {
							t.Errorf("Expected title length validation error, got %v", r.Data.Content)
						}
						return nil
					})
			},
		},
		{
			name:        "event creation error",
			interaction: createTestInteraction("Test Round", "Description", "2025-04-01 14:00", "UTC", "Location"),
			setup: func() {
				// Mock for createEvent function failing
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				// Simulate error in event creation or publication
				mockPublisher.EXPECT().
					Publish(gomock.Eq(discordroundevents.RoundCreateModalSubmit), gomock.Any()).
					Return(errors.New("failed to publish"))

				// Expect error response
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if !strings.Contains(r.Data.Content, "Round creation failed") {
							t.Errorf("Expected failure message, got %v", r.Data.Content)
						}
						return nil
					})
			},
		},
	}

	// In Test_createRoundManager_HandleCreateRoundModalSubmit:

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test expectations
			if tt.setup != nil {
				tt.setup()
			}

			// Create config with necessary values
			testConfig := &config.Config{
				Discord: config.DiscordConfig{
					GuildID: "test-guild",
				},
			}

			// Create the manager with mocked dependencies
			crm := &createRoundManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           testConfig,
				interactionStore: mockInteractionStore,
			}

			// Call the function under test with the real createEvent implementation
			crm.HandleCreateRoundModalSubmit(context.Background(), tt.interaction)
		})
	}
}

func Test_createRoundManager_HandleCreateRoundModalCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)

	// Sample interaction with member and user
	testInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID: "interaction-id",
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "user-123",
				},
			},
		},
	}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "successful_cancel",
			setup: func() {
				mockInteractionStore.EXPECT().
					Delete("interaction-id").
					Times(1)

				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...interface{}) error {
						if r.Type != discordgo.InteractionResponseChannelMessageWithSource {
							t.Errorf("Expected InteractionResponseChannelMessageWithSource, got %d", r.Type)
						}
						if !strings.Contains(r.Data.Content, "Round creation cancelled.") {
							t.Errorf("Expected cancel message, got %v", r.Data.Content)
						}
						return nil
					})
			},
		},
		{
			name: "error sending response",
			setup: func() {
				// Expect interaction store to be called
				mockInteractionStore.EXPECT().
					Delete("interaction-id").
					Times(1)

				// Expect error when sending response
				mockSession.EXPECT().
					InteractionRespond(gomock.Eq(testInteraction.Interaction), gomock.Any()).
					Return(errors.New("failed to send response"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test expectations
			if tt.setup != nil {
				tt.setup()
			}

			// Create the manager with mocked dependencies
			crm := &createRoundManager{
				session:          mockSession,
				interactionStore: mockInteractionStore,
			}

			// Call the function under test
			crm.HandleCreateRoundModalCancel(context.Background(), testInteraction)

			// Note: The function doesn't return an error, so we can't validate that directly.
			// We rely on the mock expectations to validate the behavior.
		})
	}
}
