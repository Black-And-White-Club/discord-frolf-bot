package leaderboardhandlers

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for round handlers.
type Handlers interface {
	HandleTagAssignRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagAssignedResponse(msg *message.Message) ([]*message.Message, error)
	HandleTagAssignFailedResponse(msg *message.Message) ([]*message.Message, error)
	HandleTagSwapRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagSwappedResponse(msg *message.Message) ([]*message.Message, error)
	HandleTagSwapFailedResponse(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardRetrieveRequest(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardData(msg *message.Message) ([]*message.Message, error)
	HandleGetTagByDiscordID(msg *message.Message) ([]*message.Message, error)
	HandleGetTagByDiscordIDResponse(msg *message.Message) ([]*message.Message, error)
}

// RoundHandlers handles round-related events.
type LeaderboardHandlers struct {
	Logger    observability.Logger
	Config    *config.Config
	EventUtil utils.EventUtil
	Helper    utils.Helpers
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers.

func NewLeaderboardHandlers(logger observability.Logger, config *config.Config, eventUtil utils.EventUtil, helper utils.Helpers) Handlers {
	return &LeaderboardHandlers{
		Logger:    logger,
		Config:    config,
		EventUtil: eventUtil,
		Helper:    helper,
	}
}
