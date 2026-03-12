package handlers

import (
	"context"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleChallengeFact updates Discord challenge messaging in response to backend facts.
func (h *ClubHandlers) HandleChallengeFact(ctx context.Context, topic string, payload *clubevents.ChallengeFactPayloadV1) ([]handlerwrapper.Result, error) {
	if h.challengeManager == nil || payload == nil {
		return nil, nil
	}

	if err := h.challengeManager.HandleChallengeFact(ctx, topic, payload); err != nil {
		return nil, err
	}

	return nil, nil
}
