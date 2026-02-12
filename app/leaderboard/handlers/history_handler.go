package handlers

import (
	"context"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagHistoryResponse delegates to HistoryManager.
func (h *LeaderboardHandlers) HandleTagHistoryResponse(ctx context.Context, payload *leaderboardevents.TagHistoryResponsePayloadV1) ([]handlerwrapper.Result, error) {
	if h.service == nil {
		return []handlerwrapper.Result{}, errors.New("leaderboard service is nil")
	}
	historyManager := h.service.GetHistoryManager()
	if historyManager == nil {
		return []handlerwrapper.Result{}, errors.New("history manager is nil")
	}
	historyManager.HandleTagHistoryResponse(ctx, payload)
	return []handlerwrapper.Result{}, nil
}

// HandleTagHistoryFailed delegates to HistoryManager.
func (h *LeaderboardHandlers) HandleTagHistoryFailed(ctx context.Context, payload *leaderboardevents.TagHistoryFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if h.service == nil {
		return []handlerwrapper.Result{}, errors.New("leaderboard service is nil")
	}
	historyManager := h.service.GetHistoryManager()
	if historyManager == nil {
		return []handlerwrapper.Result{}, errors.New("history manager is nil")
	}
	historyManager.HandleTagHistoryFailed(ctx, payload)
	return []handlerwrapper.Result{}, nil
}

// HandleTagGraphResponse delegates to HistoryManager.
func (h *LeaderboardHandlers) HandleTagGraphResponse(ctx context.Context, payload *leaderboardevents.TagGraphResponsePayloadV1) ([]handlerwrapper.Result, error) {
	if h.service == nil {
		return []handlerwrapper.Result{}, errors.New("leaderboard service is nil")
	}
	historyManager := h.service.GetHistoryManager()
	if historyManager == nil {
		return []handlerwrapper.Result{}, errors.New("history manager is nil")
	}
	historyManager.HandleTagGraphResponse(ctx, payload)
	return []handlerwrapper.Result{}, nil
}

// HandleTagGraphFailed delegates to HistoryManager.
func (h *LeaderboardHandlers) HandleTagGraphFailed(ctx context.Context, payload *leaderboardevents.TagGraphFailedPayloadV1) ([]handlerwrapper.Result, error) {
	if h.service == nil {
		return []handlerwrapper.Result{}, errors.New("leaderboard service is nil")
	}
	historyManager := h.service.GetHistoryManager()
	if historyManager == nil {
		return []handlerwrapper.Result{}, errors.New("history manager is nil")
	}
	historyManager.HandleTagGraphFailed(ctx, payload)
	return []handlerwrapper.Result{}, nil
}
