package dashboard

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/permission"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// FakeInteractionStore is a programmable fake for storage.ISInterface[any]
type FakeInteractionStore struct {
	*storage.FakeStorage[any]
}

func NewFakeInteractionStore() *FakeInteractionStore {
	f := &FakeInteractionStore{
		FakeStorage: storage.NewFakeStorage[any](),
	}
	f.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
		return DashboardInteractionData{
			InteractionToken: "fake-token",
			UserID:           "user-123",
			GuildID:          "guild-123",
		}, nil
	}
	return f
}

func (f *FakeInteractionStore) RecordFunc(method string) {
	// Wrapper to allow manual recording from within Funcs if needed
}

func (f *FakeInteractionStore) Calls() []string {
	return f.GetCalls()
}

// Interface assertion
var _ storage.ISInterface[any] = (*FakeInteractionStore)(nil)

// FakePermissionMapper is a programmable fake for permission.Mapper
type FakePermissionMapper struct {
	MapMemberRoleFunc func(member *discordgo.Member, guildConfig *storage.GuildConfig) permission.PWARole
	calls             []string
}

func NewFakePermissionMapper() *FakePermissionMapper {
	return &FakePermissionMapper{
		calls: []string{},
	}
}

func (f *FakePermissionMapper) record(method string) {
	f.calls = append(f.calls, method)
}

func (f *FakePermissionMapper) Calls() []string {
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *FakePermissionMapper) MapMemberRole(member *discordgo.Member, guildConfig *storage.GuildConfig) permission.PWARole {
	f.record("MapMemberRole")
	if f.MapMemberRoleFunc != nil {
		return f.MapMemberRoleFunc(member, guildConfig)
	}
	return permission.RolePlayer
}

// Interface assertion
var _ permission.Mapper = (*FakePermissionMapper)(nil)
