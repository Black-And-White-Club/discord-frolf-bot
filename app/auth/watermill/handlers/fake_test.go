package handlers

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// FakeInteractionStore is a programmable fake for storage.ISInterface[any]
type FakeInteractionStore struct {
	SetFunc    func(ctx context.Context, correlationID string, interaction any) error
	DeleteFunc func(ctx context.Context, correlationID string)
	GetFunc    func(ctx context.Context, correlationID string) (any, error)

	// Track calls for verification
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
	// Default: return valid dashboard interaction data
	return dashboard.DashboardInteractionData{
		InteractionToken: "fake-token",
		UserID:           "user-123",
		GuildID:          "guild-123",
	}, nil
}

// Interface assertion
var _ storage.ISInterface[any] = (*FakeInteractionStore)(nil)

// FakeDashboardManager is a programmable fake for dashboard.DashboardManager
type FakeDashboardManager struct {
	HandleDashboardCommandFunc func(ctx context.Context, i *discordgo.InteractionCreate) error
	calls                      []string
}

func NewFakeDashboardManager() *FakeDashboardManager {
	return &FakeDashboardManager{
		calls: []string{},
	}
}

func (f *FakeDashboardManager) record(method string) {
	f.calls = append(f.calls, method)
}

func (f *FakeDashboardManager) Calls() []string {
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *FakeDashboardManager) HandleDashboardCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	f.record("HandleDashboardCommand")
	if f.HandleDashboardCommandFunc != nil {
		return f.HandleDashboardCommandFunc(ctx, i)
	}
	return nil
}

// Interface assertion
var _ dashboard.DashboardManager = (*FakeDashboardManager)(nil)
