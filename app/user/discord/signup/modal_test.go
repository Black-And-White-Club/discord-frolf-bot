package signup

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_signupManager_SendSignupModal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := observability.NewNoOpLogger()

	tests := []struct {
		name    string
		setup   func()
		args    *discordgo.InteractionCreate
		wantErr bool
	}{
		{
			name: "successful send",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).Times(1)
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "12345"},
				},
			},
			wantErr: false,
		},

		{
			name: "failed to send modal",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("send error")).
					Times(1)
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: &discordgo.User{ID: "12345"},
				},
			},
			wantErr: true,
		},
		{
			name:    "nil interaction",
			setup:   func() {},
			args:    nil, //
			wantErr: true,
		},
		{
			name:  "nil user",
			setup: func() {},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   uuid.New().String(),
					Type: discordgo.InteractionApplicationCommand,
					User: nil,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			sm := &signupManager{
				session: mockSession,
				logger:  mockLogger,
			}

			err := sm.SendSignupModal(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendSignupModal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_signupManager_HandleSignupModalSubmit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockLogger := observability.NewNoOpLogger()

	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild_123",
		},
	}

	tests := []struct {
		name        string
		setup       func()
		args        *discordgo.InteractionCreate
		wantErr     bool
		shouldPanic bool
	}{
		{
			name: "successful submission",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any())

				mockPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			args:    validInteraction("123"),
			wantErr: false,
		},
		{
			name: "failed to acknowledge submission",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("acknowledge error"))
			},
			args:    validInteraction("123"),
			wantErr: true,
		},
		{
			name: "failed to store interaction",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_, _, _ any) {
						panic("cache error")
					})
			},
			args:        validInteraction("123"),
			wantErr:     true,
			shouldPanic: true,
		},
		{
			name: "failed to publish event",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				mockInteractionStore.EXPECT().
					Set(gomock.Any(), gomock.Any(), gomock.Any())

				mockPublisher.EXPECT().
					Publish(gomock.Any(), gomock.Any()).
					Return(errors.New("publish error"))
			},
			args:    validInteraction("123"),
			wantErr: true,
		},
		{
			name: "missing Interaction.ID",
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    "",
					Token: uuid.New().String(),
					Type:  discordgo.InteractionModalSubmit,
					Data: &discordgo.ModalSubmitInteractionData{
						CustomID: "signup_modal",
					},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "12345"},
					},
				},
			},
			wantErr: true, // Expecting an error or early return
		},

		{
			name: "missing Interaction.Token",
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    uuid.New().String(),
					Token: "",
					Type:  discordgo.InteractionModalSubmit,
					Data: &discordgo.ModalSubmitInteractionData{
						CustomID: "signup_modal",
					},
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "12345"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing Interaction.Data",
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    uuid.New().String(),
					Token: uuid.New().String(),
					Type:  discordgo.InteractionModalSubmit,
					Data:  nil,
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "12345"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "nil interaction",
			setup: func() {
			},
			args:    nil,
			wantErr: true,
		},

		{
			name: "nil user in member",
			setup: func() {
			},
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:    uuid.New().String(),
					Token: uuid.New().String(),
					Type:  discordgo.InteractionModalSubmit,
					Member: &discordgo.Member{
						User: nil,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			sm := &signupManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				interactionStore: mockInteractionStore,
				config:           mockConfig,
			}

			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("Test %s panicked unexpectedly: %v", tt.name, r)
					}
				} else if tt.shouldPanic {
					t.Errorf("Expected panic, but test %s did not panic", tt.name)
				}
			}()

			sm.HandleSignupModalSubmit(context.Background(), tt.args)
		})
	}
}

func validInteraction(tagValue string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:    uuid.New().String(),
			Token: uuid.New().String(),
			Type:  discordgo.InteractionModalSubmit,
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: "signup_modal",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "tag_number",
								Value:    tagValue, // Customizable value for testing
							},
						},
					},
				},
			},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "12345"},
			},
		},
	}
}
