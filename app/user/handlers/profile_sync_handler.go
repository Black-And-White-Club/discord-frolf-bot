package handlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleProfileSyncRequest handles requests to sync a user's profile from Discord.
// This is triggered when the backend detects missing display name data in club memberships.
//
// Flow:
//  1. frolf-bot publishes user.profile.sync.request.v1 when GetTicket detects empty display name
//  2. discord-frolf-bot receives request and fetches member from Discord API
//  3. discord-frolf-bot publishes user.profile.updated.v1 with fresh profile data
//  4. frolf-bot receives update and persists to club_memberships table
func (h *UserHandlers) HandleProfileSyncRequest(
	ctx context.Context,
	payload *userevents.UserProfileSyncRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling profile sync request",
		attr.String("user_id", string(payload.UserID)),
		attr.String("guild_id", string(payload.GuildID)),
	)

	err := h.service.SyncGuildMember(ctx, string(payload.GuildID), string(payload.UserID))
	if err != nil {
		h.logger.WarnContext(ctx, "Profile sync failed",
			attr.Error(err),
			attr.String("user_id", string(payload.UserID)),
			attr.String("guild_id", string(payload.GuildID)),
		)
		// Don't return error - this is a best-effort sync, failures are logged but don't block
		return nil, nil
	}

	h.logger.InfoContext(ctx, "Profile sync completed successfully",
		attr.String("user_id", string(payload.UserID)),
		attr.String("guild_id", string(payload.GuildID)),
	)

	// No downstream events needed - the SyncGuildMember already publishes UserProfileUpdatedV1
	return nil, nil
}
