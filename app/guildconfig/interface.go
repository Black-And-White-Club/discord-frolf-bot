package guildconfig

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
)

// GuildConfigResolver defines the interface for guild config resolution and caching.
type GuildConfigResolver interface {
	GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	RequestGuildConfigAsync(ctx context.Context, guildID string)
	IsGuildSetupComplete(guildID string) bool
	HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig)
	HandleBackendError(ctx context.Context, guildID string, err error)
	ClearInflightRequest(ctx context.Context, guildID string)
}
