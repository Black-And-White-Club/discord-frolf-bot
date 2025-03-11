package scorehandlers

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Handlers interface {
	HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateResponse(msg *message.Message) ([]*message.Message, error)
}

// RoundHandlers handles round-related events.
type ScoreHandlers struct {
	Logger    observability.Logger
	Session   discord.Session
	Config    *config.Config
	EventUtil utils.EventUtil
}

// NewScoreHandlers creates a new ScoreHandlers.

func NewScoreHandlers(logger observability.Logger, session discord.Session, config *config.Config, eventUtil utils.EventUtil) Handlers {
	return &ScoreHandlers{
		Logger:    logger,
		Session:   session,
		Config:    config,
		EventUtil: eventUtil,
	}
}
