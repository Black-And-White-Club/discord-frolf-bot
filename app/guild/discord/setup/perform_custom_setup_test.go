package setup

import (
	"context"
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_performCustomSetup(t *testing.T) {
	baseGuild := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{{Name: "Existing", ID: "Existing-id"}}}

	tests := []struct {
		name    string
		cfg     SetupConfig
		setup   func(m *discordmocks.MockSession)
		wantErr bool
		check   func(t *testing.T, res *SetupResult, err error)
	}{
		{
			name: "creates channels roles and signup message (happy path)",
			cfg:  SetupConfig{ChannelPrefix: "frolf", UserRoleName: "Player", EditorRoleName: "Editor", AdminRoleName: "Admin", SignupMessage: "Hi", SignupEmoji: "🥏", CreateChannels: true, CreateRoles: true, CreateSignupMsg: true},
			setup: func(ms *discordmocks.MockSession) {
				ms.EXPECT().Guild("g1", gomock.Any()).Return(baseGuild, nil)
				ms.EXPECT().GuildChannels("g1", gomock.Any()).Return([]*discordgo.Channel{}, nil).AnyTimes()
				ms.EXPECT().GuildChannelCreate("g1", "frolf-events", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "events-id"}, nil)
				ms.EXPECT().ChannelEdit("events-id", gomock.Any(), gomock.Any()).Return(&discordgo.Channel{ID: "events-id"}, nil).AnyTimes()
				ms.EXPECT().GuildChannelCreate("g1", "frolf-leaderboard", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "leaderboard-id"}, nil)
				ms.EXPECT().GuildChannelCreate("g1", "frolf-signup", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "signup-id"}, nil)
				ms.EXPECT().GuildRoleCreate("g1", gomock.Any(), gomock.Any()).Return(&discordgo.Role{Name: "Player", ID: "Player-id"}, nil)
				ms.EXPECT().GuildRoleCreate("g1", gomock.Any(), gomock.Any()).Return(&discordgo.Role{Name: "Editor", ID: "Editor-id"}, nil)
				ms.EXPECT().GuildRoleCreate("g1", gomock.Any(), gomock.Any()).Return(&discordgo.Role{Name: "Admin", ID: "Admin-id"}, nil)
				ms.EXPECT().ChannelMessageSend("signup-id", gomock.Any(), gomock.Any()).Return(&discordgo.Message{ID: "m1"}, nil)
				ms.EXPECT().MessageReactionAdd("signup-id", "m1", "🥏").Return(nil)
			},
			check: func(t *testing.T, res *SetupResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.UserRoleID == "" || res.AdminRoleID == "" {
					t.Errorf("expected roles created")
				}
				if res.SignupMessageID == "" {
					t.Errorf("expected signup message created")
				}
			},
		},
		{
			name:    "guild fetch error",
			cfg:     SetupConfig{CreateChannels: true},
			wantErr: true,
			setup: func(ms *discordmocks.MockSession) {
				ms.EXPECT().Guild("g1", gomock.Any()).Return(nil, errors.New("no guild"))
			},
		},
		{
			name:    "channel creation failure aborts",
			cfg:     SetupConfig{ChannelPrefix: "frolf", CreateChannels: true},
			wantErr: true,
			setup: func(ms *discordmocks.MockSession) {
				ms.EXPECT().Guild("g1", gomock.Any()).Return(baseGuild, nil)
				ms.EXPECT().GuildChannels("g1", gomock.Any()).Return([]*discordgo.Channel{}, nil).AnyTimes()
				ms.EXPECT().GuildChannelCreate("g1", "frolf-events", discordgo.ChannelTypeGuildText, gomock.Any()).Return(nil, errors.New("fail ch"))
			},
		},
		{
			name:    "role creation empty ID error",
			cfg:     SetupConfig{ChannelPrefix: "frolf", UserRoleName: "Player", EditorRoleName: "Editor", AdminRoleName: "Admin", CreateRoles: true},
			wantErr: true,
			setup: func(ms *discordmocks.MockSession) {
				ms.EXPECT().Guild("g1", gomock.Any()).Return(&discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{}}, nil)
				ms.EXPECT().GuildRoleCreate("g1", gomock.Any(), gomock.Any()).Return(&discordgo.Role{Name: "Player", ID: ""}, nil)
			},
		},
		{
			name:    "signup message error surfaces",
			cfg:     SetupConfig{ChannelPrefix: "frolf", CreateChannels: true, CreateSignupMsg: true},
			wantErr: true,
			setup: func(ms *discordmocks.MockSession) {
				ms.EXPECT().Guild("g1", gomock.Any()).Return(baseGuild, nil)
				ms.EXPECT().GuildChannels("g1", gomock.Any()).Return([]*discordgo.Channel{}, nil).AnyTimes()
				ms.EXPECT().GuildChannelCreate("g1", "frolf-events", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "events-id"}, nil)
				ms.EXPECT().ChannelEdit("events-id", gomock.Any(), gomock.Any()).Return(&discordgo.Channel{ID: "events-id"}, nil).AnyTimes()
				ms.EXPECT().GuildChannelCreate("g1", "frolf-leaderboard", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "leaderboard-id"}, nil)
				ms.EXPECT().GuildChannelCreate("g1", "frolf-signup", discordgo.ChannelTypeGuildText, gomock.Any()).Return(&discordgo.Channel{ID: "signup-id"}, nil)
				ms.EXPECT().ChannelMessageSend("signup-id", gomock.Any(), gomock.Any()).Return(nil, errors.New("msg fail"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ms := discordmocks.NewMockSession(ctrl)
			if tt.setup != nil {
				tt.setup(ms)
			}
			m := &setupManager{session: ms}
			res, err := m.performCustomSetup(context.Background(), "g1", tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, res, err)
			}
		})
	}
}
