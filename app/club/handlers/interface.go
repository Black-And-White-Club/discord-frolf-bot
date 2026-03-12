package handlers

import (
	"context"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the contract for club event handlers.
type Handlers interface {
	HandleChallengeFact(ctx context.Context, topic string, payload *clubevents.ChallengeFactPayloadV1) ([]handlerwrapper.Result, error)
}
