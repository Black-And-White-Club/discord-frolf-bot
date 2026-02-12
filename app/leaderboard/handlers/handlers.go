package handlers

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for leaderboard handlers.
// All handlers use pure transformation pattern: context + typed payload → Results.
type Handlers interface {
	// Leaderboard Retrieval
	HandleLeaderboardRetrieveRequest(ctx context.Context, payload *discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleLeaderboardUpdatedNotification(ctx context.Context, payload *leaderboardevents.LeaderboardUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleLeaderboardResponse(ctx context.Context, payload *leaderboardevents.GetLeaderboardResponsePayloadV1) ([]handlerwrapper.Result, error)

	// Leaderboard Errors
	HandleLeaderboardUpdateFailed(ctx context.Context, payload *leaderboardevents.LeaderboardUpdateFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleLeaderboardRetrievalFailed(ctx context.Context, payload *leaderboardevents.GetLeaderboardFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Tag Number Lookups
	HandleGetTagByDiscordID(ctx context.Context, payload *discordleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetTagByDiscordIDResponse(ctx context.Context, payload *sharedevents.GetTagNumberResponsePayloadV1) ([]handlerwrapper.Result, error)
	HandleGetTagByDiscordIDFailed(ctx context.Context, payload *sharedevents.GetTagNumberFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Tag Assignment
	HandleTagAssignRequest(ctx context.Context, payload *discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagAssignedResponse(ctx context.Context, payload *leaderboardevents.LeaderboardTagAssignedPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagAssignFailedResponse(ctx context.Context, payload *leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Tag Swap
	HandleTagSwapRequest(ctx context.Context, payload *discordleaderboardevents.LeaderboardTagSwapRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagSwappedResponse(ctx context.Context, payload *leaderboardevents.TagSwapProcessedPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagSwapFailedResponse(ctx context.Context, payload *leaderboardevents.TagSwapFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Leaderboard Updates
	HandleBatchTagAssigned(ctx context.Context, payload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1) ([]handlerwrapper.Result, error)

	// Season Management
	HandleSeasonStartedResponse(ctx context.Context, payload *leaderboardevents.StartNewSeasonSuccessPayloadV1) ([]handlerwrapper.Result, error)
	HandleSeasonStartFailedResponse(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetSeasonStandingsResponse(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1) ([]handlerwrapper.Result, error)
	HandleGetSeasonStandingsFailedResponse(ctx context.Context, payload *leaderboardevents.AdminFailedPayloadV1) ([]handlerwrapper.Result, error)
}

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	service             leaderboarddiscord.LeaderboardDiscordInterface
	helpers             utils.Helpers
	config              *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	logger              *slog.Logger
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	leaderboardDiscord leaderboarddiscord.LeaderboardDiscordInterface,
	guildConfigResolver guildconfig.GuildConfigResolver,
) Handlers {
	return &LeaderboardHandlers{
		service:             leaderboardDiscord,
		helpers:             helpers,
		config:              config,
		guildConfigResolver: guildConfigResolver,
		logger:              logger,
	}
}
