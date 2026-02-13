package handlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleSeasonStartedResponse handles a successful season start event from the backend.
func (h *LeaderboardHandlers) HandleSeasonStartedResponse(ctx context.Context,
	payload *leaderboardevents.StartNewSeasonSuccessPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling season started response",
		attr.String("season_id", payload.SeasonID),
		attr.String("guild_id", string(payload.GuildID)))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonStarted(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}

// HandleSeasonStartFailedResponse handles a failed season start event from the backend.
func (h *LeaderboardHandlers) HandleSeasonStartFailedResponse(ctx context.Context,
	payload *leaderboardevents.AdminFailedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.WarnContext(ctx, "Handling season start failed response",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonStartFailed(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}

// HandleGetSeasonStandingsResponse handles a successful season standings response from the backend.
func (h *LeaderboardHandlers) HandleGetSeasonStandingsResponse(ctx context.Context,
	payload *leaderboardevents.GetSeasonStandingsResponsePayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling season standings response",
		attr.String("season_id", payload.SeasonID),
		attr.String("guild_id", string(payload.GuildID)))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonStandings(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}

// HandleGetSeasonStandingsFailedResponse handles a failed season standings retrieval from the backend.
func (h *LeaderboardHandlers) HandleGetSeasonStandingsFailedResponse(ctx context.Context,
	payload *leaderboardevents.AdminFailedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.WarnContext(ctx, "Handling season standings failed response",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonStandingsFailed(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}

// HandleSeasonEndedResponse handles a successful season end event from the backend.
func (h *LeaderboardHandlers) HandleSeasonEndedResponse(ctx context.Context,
	payload *leaderboardevents.EndSeasonSuccessPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling season ended response",
		attr.String("guild_id", string(payload.GuildID)))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonEnded(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}

// HandleSeasonEndFailedResponse handles a failed season end event from the backend.
func (h *LeaderboardHandlers) HandleSeasonEndFailedResponse(ctx context.Context,
	payload *leaderboardevents.AdminFailedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.WarnContext(ctx, "Handling season end failed response",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("reason", payload.Reason))

	if h.service != nil {
		seasonManager := h.service.GetSeasonManager()
		if seasonManager != nil {
			seasonManager.HandleSeasonEndFailed(ctx, payload)
		}
	}

	return []handlerwrapper.Result{}, nil
}
