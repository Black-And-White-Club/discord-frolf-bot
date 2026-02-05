package handlers

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	"github.com/bwmarrin/discordgo"
)

// FakeInteractionStore is a programmable fake for storage.ISInterface[any]
type FakeInteractionStore struct {
	*testutils.FakeStorage[any]
}

func NewFakeInteractionStore() *FakeInteractionStore {
	f := &FakeInteractionStore{
		FakeStorage: testutils.NewFakeStorage[any](),
	}
	f.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
		return dashboard.DashboardInteractionData{
			InteractionToken: "fake-token",
			UserID:           "user-123",
			GuildID:          "guild-123",
		}, nil
	}
	return f
}

func (f *FakeInteractionStore) Calls() []string {
	return f.GetCalls()
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
