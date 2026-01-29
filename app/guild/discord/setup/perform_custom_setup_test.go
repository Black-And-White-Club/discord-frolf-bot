package setup

import (
	"context"
	"errors"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

func Test_performCustomSetup(t *testing.T) {
	baseGuild := &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{{Name: "Existing", ID: "Existing-id"}}}

	tests := []struct {
		name    string
		cfg     SetupConfig
		setup   func(m *discord.FakeSession)
		wantErr bool
		check   func(t *testing.T, res *SetupResult, err error)
	}{
		{
			name: "creates channels roles and signup message (happy path)",
			cfg:  SetupConfig{ChannelPrefix: "frolf", UserRoleName: "Player", EditorRoleName: "Editor", AdminRoleName: "Admin", SignupMessage: "Hi", SignupEmoji: "🥏", CreateChannels: true, CreateRoles: true, CreateSignupMsg: true},
			setup: func(ms *discord.FakeSession) {
				ms.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
					return baseGuild, nil
				}
				ms.GuildChannelsFunc = func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
					return []*discordgo.Channel{}, nil
				}
				ms.GuildChannelCreateFunc = func(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					switch name {
					case "frolf-events":
						return &discordgo.Channel{ID: "events-id"}, nil
					case "frolf-leaderboard":
						return &discordgo.Channel{ID: "leaderboard-id"}, nil
					case "frolf-signup":
						return &discordgo.Channel{ID: "signup-id"}, nil
					}
					return &discordgo.Channel{ID: "id"}, nil
				}
				ms.ChannelEditFunc = func(channelID string, data *discordgo.ChannelEdit, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: channelID}, nil
				}
				ms.GuildRoleCreateFunc = func(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
					return &discordgo.Role{Name: params.Name, ID: params.Name + "-id"}, nil
				}
				ms.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "m1"}, nil
				}
				ms.MessageReactionAddFunc = func(channelID, messageID, emojiID string) error {
					return nil
				}
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
			setup: func(ms *discord.FakeSession) {
				ms.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
					return nil, errors.New("no guild")
				}
			},
		},
		{
			name:    "channel creation failure aborts",
			cfg:     SetupConfig{ChannelPrefix: "frolf", CreateChannels: true},
			wantErr: true,
			setup: func(ms *discord.FakeSession) {
				ms.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
					return baseGuild, nil
				}
				ms.GuildChannelsFunc = func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
					return []*discordgo.Channel{}, nil
				}
				ms.GuildChannelCreateFunc = func(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("fail ch")
				}
			},
		},
		{
			name:    "role creation empty ID error",
			cfg:     SetupConfig{ChannelPrefix: "frolf", UserRoleName: "Player", EditorRoleName: "Editor", AdminRoleName: "Admin", CreateRoles: true},
			wantErr: true,
			setup: func(ms *discord.FakeSession) {
				ms.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
					return &discordgo.Guild{ID: "g1", Roles: []*discordgo.Role{}}, nil
				}
				ms.GuildRoleCreateFunc = func(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
					return &discordgo.Role{Name: "Player", ID: ""}, nil
				}
			},
		},
		{
			name:    "signup message error surfaces",
			cfg:     SetupConfig{ChannelPrefix: "frolf", CreateChannels: true, CreateSignupMsg: true},
			wantErr: true,
			setup: func(ms *discord.FakeSession) {
				ms.GuildFunc = func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
					return baseGuild, nil
				}
				ms.GuildChannelsFunc = func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
					return []*discordgo.Channel{}, nil
				}
				ms.GuildChannelCreateFunc = func(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return &discordgo.Channel{ID: "signup-id"}, nil
				}
				ms.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("msg fail")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := discord.NewFakeSession()
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
