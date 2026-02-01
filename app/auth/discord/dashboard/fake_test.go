package dashboard

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/permission"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// FakeInteractionStore is a programmable fake for storage.ISInterface[any]
type FakeInteractionStore struct {
	SetFunc    func(ctx context.Context, correlationID string, interaction any) error
	DeleteFunc func(ctx context.Context, correlationID string)
	GetFunc    func(ctx context.Context, correlationID string) (any, error)

	calls []string
}

func NewFakeInteractionStore() *FakeInteractionStore {
	return &FakeInteractionStore{
		calls: []string{},
	}
}

func (f *FakeInteractionStore) record(method string) {
	f.calls = append(f.calls, method)
}

func (f *FakeInteractionStore) Calls() []string {
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *FakeInteractionStore) Set(ctx context.Context, correlationID string, interaction any) error {
	f.record("Set")
	if f.SetFunc != nil {
		return f.SetFunc(ctx, correlationID, interaction)
	}
	return nil
}

func (f *FakeInteractionStore) Delete(ctx context.Context, correlationID string) {
	f.record("Delete")
	if f.DeleteFunc != nil {
		f.DeleteFunc(ctx, correlationID)
	}
}

func (f *FakeInteractionStore) Get(ctx context.Context, correlationID string) (any, error) {
	f.record("Get")
	if f.GetFunc != nil {
		return f.GetFunc(ctx, correlationID)
	}
	return DashboardInteractionData{
		InteractionToken: "fake-token",
		UserID:           "user-123",
		GuildID:          "guild-123",
	}, nil
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
