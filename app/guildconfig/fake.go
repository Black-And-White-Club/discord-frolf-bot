package guildconfig

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
)

// FakeGuildConfigResolver provides a programmable stub for the GuildConfigResolver interface.
type FakeGuildConfigResolver struct {
	GetGuildConfigWithContextFunc func(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	RequestGuildConfigAsyncFunc   func(ctx context.Context, guildID string)
	IsGuildSetupCompleteFunc      func(guildID string) bool
	HandleGuildConfigReceivedFunc func(ctx context.Context, guildID string, config *storage.GuildConfig)
	HandleBackendErrorFunc        func(ctx context.Context, guildID string, err error)
	ClearInflightRequestFunc      func(ctx context.Context, guildID string)
}

func (f *FakeGuildConfigResolver) GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
	if f.GetGuildConfigWithContextFunc != nil {
		return f.GetGuildConfigWithContextFunc(ctx, guildID)
	}
	return nil, nil
}

func (f *FakeGuildConfigResolver) RequestGuildConfigAsync(ctx context.Context, guildID string) {
	if f.RequestGuildConfigAsyncFunc != nil {
		f.RequestGuildConfigAsyncFunc(ctx, guildID)
	}
}

func (f *FakeGuildConfigResolver) IsGuildSetupComplete(guildID string) bool {
	if f.IsGuildSetupCompleteFunc != nil {
		return f.IsGuildSetupCompleteFunc(guildID)
	}
	return false
}

func (f *FakeGuildConfigResolver) HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig) {
	if f.HandleGuildConfigReceivedFunc != nil {
		f.HandleGuildConfigReceivedFunc(ctx, guildID, config)
	}
}

func (f *FakeGuildConfigResolver) HandleBackendError(ctx context.Context, guildID string, err error) {
	if f.HandleBackendErrorFunc != nil {
		f.HandleBackendErrorFunc(ctx, guildID, err)
	}
}

func (f *FakeGuildConfigResolver) ClearInflightRequest(ctx context.Context, guildID string) {
	if f.ClearInflightRequestFunc != nil {
		f.ClearInflightRequestFunc(ctx, guildID)
	}
}

var _ GuildConfigResolver = (*FakeGuildConfigResolver)(nil)
