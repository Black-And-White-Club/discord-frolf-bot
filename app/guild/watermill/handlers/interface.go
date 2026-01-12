package handlers

import (
	"context"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the contract for guild event handlers following the pure transformation pattern.
type Handlers interface {
	// Guild config creation/setup handlers
	HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigCreationFailed(ctx context.Context, payload *guildevents.GuildConfigCreationFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Guild config update handlers
	HandleGuildConfigUpdated(ctx context.Context, payload *guildevents.GuildConfigUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigUpdateFailed(ctx context.Context, payload *guildevents.GuildConfigUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigRetrieved(ctx context.Context, payload *guildevents.GuildConfigRetrievedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigRetrievalFailed(ctx context.Context, payload *guildevents.GuildConfigRetrievalFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Guild config deletion handlers
	HandleGuildConfigDeleted(ctx context.Context, payload *guildevents.GuildConfigDeletedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigDeletionFailed(ctx context.Context, payload *guildevents.GuildConfigDeletionFailedPayloadV1) ([]handlerwrapper.Result, error)
}
