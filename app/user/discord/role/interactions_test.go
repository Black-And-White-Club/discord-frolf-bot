package role

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	util_mocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_roleManager_RespondToRoleRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)

	tests := []struct {
		name             string
		setup            func()
		ctx              context.Context
		interactionID    string
		interactionToken string
		targetUserID     string
		wantErr          bool
	}{
		{
			name: "successful role request response",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			targetUserID:     "target-user-id",
			wantErr:          false,
		},
		{
			name: "failed to respond to role request",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("respond to role request error")).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			targetUserID:     "target-user-id",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			err := rm.RespondToRoleRequest(tt.ctx, tt.interactionID, tt.interactionToken, tt.targetUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("roleManager.RespondToRoleRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_roleManager_RespondToRoleButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()

	tests := []struct {
		name             string
		setup            func()
		interactionID    string
		interactionToken string
		requesterID      string
		selectedRole     string
		targetUserID     string
		wantErr          bool
	}{
		{
			name: "successful button press response",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			requesterID:      "requester-id",
			selectedRole:     "Admin",
			targetUserID:     "target-user-id",
			wantErr:          false,
		},
		{
			name: "failed to acknowledge button press",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).Return(errors.New("acknowledge error")).
					Times(1)
			},
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			requesterID:      "requester-id",
			selectedRole:     "User ",
			targetUserID:     "target-user-id",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:   mockSession,
				publisher: mockPublisher,
				logger:    mockLogger,
			}

			err := rm.RespondToRoleButtonPress(context.Background(), tt.interactionID, tt.interactionToken, tt.requesterID, tt.selectedRole, tt.targetUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("roleManager.RespondToRoleButtonPress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_roleManager_HandleRoleRequestCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)

	tests := []struct {
		name    string
		setup   func()
		ctx     context.Context
		i       *discordgo.InteractionCreate
		wantErr bool
	}{
		{
			name:    "nil interaction",
			setup:   func() {},
			ctx:     context.Background(),
			i:       nil,
			wantErr: false,
		},
		{
			name:  "nil interaction interaction",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: nil,
			},
			wantErr: false,
		},
		{
			name:  "nil user",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					User: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "successful role request handling",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to respond to role request",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("respond to role request error")).
					Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			rm.HandleRoleRequestCommand(tt.ctx, tt.i)
		})
	}
}

func Test_roleManager_HandleRoleButtonPress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)

	tests := []struct {
		name    string
		setup   func()
		ctx     context.Context
		i       *discordgo.InteractionCreate
		wantErr bool
	}{
		{
			name:    "nil interaction",
			setup:   func() {},
			ctx:     context.Background(),
			i:       nil,
			wantErr: false,
		},
		{
			name:  "nil interaction interaction",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: nil,
			},
			wantErr: false,
		},
		{
			name:  "unexpected interaction data type",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
					Data: &discordgo.ApplicationCommandInteractionData{}, // Wrong type
				},
			},
			wantErr: false,
		},
		{
			name:  "no mentions in message",
			setup: func() {},
			ctx:   context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:   "interaction-id",
					User: &discordgo.User{ID: "user-id"},
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{}, // No mentions
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid role button press",
			setup: func() {
				// mockPublisher.EXPECT().
				// 	Publish(gomock.Any(), gomock.Any()).
				// 	Return(nil).
				// 	Times(1) // Ensure the event is published exactly once

				// mockInteractionStore.EXPECT().
				// 	Set(gomock.Any(), gomock.Any(), gomock.Any()).
				// 	Times(1) // Ensure the interaction is stored
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid role button press",
			setup: func() {
				// mockPublisher.EXPECT().
				// 	Publish(gomock.Any(), gomock.Any()).
				// 	Do(func(event string, msg *message.Message) {
				// 		fmt.Println("Publish() called with event:", event)
				// 	}).
				// 	Return(nil).
				// 	Times(1)

				// mockInteractionStore.EXPECT().
				// 	Set(gomock.Any(), gomock.Any(), gomock.Any()).
				// 	Do(func(key string, val interface{}, ttl time.Duration) {
				// 		fmt.Println("Set() called with key:", key)
				// 	}).
				// 	Times(1)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in storing interaction reference in cache",
			setup: func() {
				mockInteractionStore.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("error storing interaction reference"))
				// Do not expect Publish to be called
				mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(0)
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-id"},
					},
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in publishing event to JetStream",
			setup: func() {
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error in sending error response to user",
			setup: func() {
			},
			ctx: context.Background(),
			i: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      "interaction-id",
					User:    &discordgo.User{ID: "user-id"},
					Token:   "interaction-token",
					GuildID: "guild-id",
					Data: &discordgo.MessageComponentInteractionData{
						CustomID: "role_button_admin",
					},
					Message: &discordgo.Message{
						Mentions: []*discordgo.User{
							{ID: "mentioned-user-id"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			rm.HandleRoleButtonPress(tt.ctx, tt.i)
		})
	}
}

func Test_roleManager_HandleRoleCancelButton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)

	tests := []struct {
		name             string
		setup            func()
		ctx              context.Context
		interactionID    string
		interactionToken string
		wantErr          bool
	}{
		{
			name: "successful role cancel button handling",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "failed to respond to interaction",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(errors.New("respond to interaction error")).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "failed to delete interaction from store",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(1)
			},
			ctx:              context.Background(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "nil_interaction",
			setup: func() {
				// Do not expect any interactions
			},
			ctx:              context.Background(),
			interactionID:    "",
			interactionToken: "",
			wantErr:          false,
		},
		{
			name: "empty interaction ID",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(0)
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(0)
			},
			ctx:              context.Background(),
			interactionID:    "",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
		{
			name: "cancelled_context",
			setup: func() {
				mockSession.EXPECT().
					InteractionRespond(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(0) // No response should be sent
				mockInteractionStore.EXPECT().
					Delete(gomock.Any()).
					Times(0) // No deletion should occur
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Immediately cancel the context
				return ctx
			}(),
			interactionID:    "interaction-id",
			interactionToken: "interaction-token",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rm := &roleManager{
				session:          mockSession,
				publisher:        mockPublisher,
				logger:           mockLogger,
				helper:           mockHelper,
				config:           mockConfig,
				interactionStore: mockInteractionStore,
			}

			var interaction *discordgo.InteractionCreate
			if tt.interactionID != "" {
				interaction = &discordgo.InteractionCreate{
					Interaction: &discordgo.Interaction{
						ID: tt.interactionID,
					},
				}
			}

			if interaction != nil {
				rm.HandleRoleCancelButton(tt.ctx, interaction)
			} else {
				rm.HandleRoleCancelButton(tt.ctx, nil)
			}
		})
	}
}
