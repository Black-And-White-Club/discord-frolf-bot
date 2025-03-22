package leaderboardhandlers

import (
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for leaderboard handlers.
type Handlers interface {
	HandleLeaderboardUpdated(msg *message.Message) ([]*message.Message, error)
}

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	Logger             observability.Logger
	Config             *config.Config
	EventUtil          utils.EventUtil
	Helpers            utils.Helpers
	LeaderboardDiscord leaderboarddiscord.LeaderboardDiscordInterface
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(logger observability.Logger, config *config.Config, eventUtil utils.EventUtil, helpers utils.Helpers, leaderboardDiscord leaderboarddiscord.LeaderboardDiscordInterface) Handlers {
	return &LeaderboardHandlers{
		Logger:             logger,
		Config:             config,
		EventUtil:          eventUtil,
		Helpers:            helpers,
		LeaderboardDiscord: leaderboardDiscord,
	}
}
