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

func Test_roleManager_EditRoleUpdateResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()
	mockConfig := &config.Config{}
	mockInteractionStore := storagemocks.NewMockISInterface(ctrl)
	mockHelper := util_mocks.NewMockHelpers(ctrl)

	tests := []struct {
		name          string
		setup         func()
		ctx           context.Context
		correlationID string
		content       string
		wantErr       bool
	}{
		{
			name: "successful role update response edit",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get(gomock.Any()).
					Return(&discordgo.Interaction{ID: "interaction-id"}, true).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					Return(nil, nil).
					Times(1)
			},
			ctx:           context.Background(),
			correlationID: "correlation-id",
			content:       "Role updated successfully.",
			wantErr:       false,
		},
		{
			name: "failed to get interaction from store",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get(gomock.Any()).
					Return(nil, false).
					Times(1)
			},
			ctx:           context.Background(),
			correlationID: "correlation-id",
			content:       "Role updated successfully.",
			wantErr:       true,
		},
		{
			name: "stored interaction is not of the expected type",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get(gomock.Any()).
					Return("not an interaction", true).
					Times(1)
			},
			ctx:           context.Background(),
			correlationID: "correlation-id",
			content:       "Role updated successfully.",
			wantErr:       true,
		},
		{
			name: "failed to edit interaction response",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get(gomock.Any()).
					Return(&discordgo.Interaction{ID: "interaction-id"}, true).
					Times(1)
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					Return(&discordgo.Message{}, errors.New("edit interaction response error")).
					Times(1)
			},
			ctx:           context.Background(),
			correlationID: "correlation-id",
			content:       "Role updated successfully.",
			wantErr:       true,
		},
		{
			name: "cancelled_context",
			setup: func() {
				mockInteractionStore.EXPECT().
					Get(gomock.Any()).
					Times(0) // Do not expect any calls to Get
				mockSession.EXPECT().
					InteractionResponseEdit(gomock.Any(), gomock.Any()).
					Times(0) // Do not expect any calls to InteractionResponseEdit
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			correlationID: "correlation-id",
			content:       "Role updated successfully.",
			wantErr:       true,
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

			if err := rm.EditRoleUpdateResponse(tt.ctx, tt.correlationID, tt.content); (err != nil) != tt.wantErr {
				t.Errorf("roleManager.EditRoleUpdateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_roleManager_AddRoleToUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)
	mockLogger := observability.NewNoOpLogger()

	mockConfig := &config.Config{}

	tests := []struct {
		name    string
		setup   func()
		guildID string
		userID  string
		roleID  string
		wantErr bool
	}{
		{
			name: "successful role addition",
			setup: func() {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-id", "user-id", "role-id").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-id", "user-id").
					Return(&discordgo.Member{Roles: []string{"role-id"}}, nil).
					Times(1)
			},
			guildID: "guild-id",
			userID:  "user-id",
			roleID:  "role-id",
			wantErr: false,
		},
		{
			name: "failed to add role",
			setup: func() {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-id", "user-id", "role-id").
					Return(errors.New("add role error")).
					Times(1)
			},
			guildID: "guild-id",
			userID:  "user-id",
			roleID:  "role-id",
			wantErr: true,
		},
		{
			name: "failed to fetch user after adding role",
			setup: func() {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-id", "user-id", "role-id").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-id", "user-id").
					Return(nil, errors.New("fetch user error")).
					Times(1)
			},
			guildID: "guild-id",
			userID:  "user-id",
			roleID:  "role-id",
			wantErr: true,
		},
		{
			name: "role not added to user",
			setup: func() {
				mockSession.EXPECT().
					GuildMemberRoleAdd("guild-id", "user-id", "role-id").
					Return(nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember("guild-id", "user-id").
					Return(&discordgo.Member{Roles: []string{}}, nil).
					Times(1)
			},
			guildID: "guild-id",
			userID:  "user-id",
			roleID:  "role-id",
			wantErr: true,
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
				config:    mockConfig,
			}

			err := rm.AddRoleToUser(context.Background(), tt.guildID, tt.userID, tt.roleID)
			if (err != nil) != tt.wantErr {
				t.Errorf("roleManager.AddRoleToUser () error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
