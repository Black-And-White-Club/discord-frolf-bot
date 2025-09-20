package guildconfig

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
)

// GuildConfigResolver defines the interface for guild config resolution and caching.
type GuildConfigResolver interface {
	// GetGuildConfigWithContext retrieves the guild config for a given guild ID with context support.
	GetGuildConfigWithContext(ctx context.Context, guildID string) (*storage.GuildConfig, error)
	// IsGuildSetupComplete checks if a guild has completed setup (backend fetch each call).
	IsGuildSetupComplete(guildID string) bool
	// HandleGuildConfigReceived processes config responses from backend and notifies waiters.
	HandleGuildConfigReceived(ctx context.Context, guildID string, config *storage.GuildConfig)
}
