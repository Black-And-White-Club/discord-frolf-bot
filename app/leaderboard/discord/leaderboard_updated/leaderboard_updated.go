package leaderboardupdated

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// LeaderboardUpdateManager defines the interface for create round operations.
type LeaderboardUpdateManager interface {
	HandleLeaderboardPagination(ctx context.Context, i *discordgo.InteractionCreate)
	SendLeaderboardEmbed(channelID string, leaderboard []LeaderboardEntry, page int) (*discordgo.Message, error)
}

// leaderboardUpdateManager implements the LeaderboardUpdateManager interface.
type leaderboardUpdateManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewCreateLeaderboardUpdateManager creates a new CreateLeaderboardUpdateManager instance.
func NewCreateLeaderboardUpdateManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config) LeaderboardUpdateManager {
	logger.Info(context.Background(), "Creating CreateLeaderboardUpdateManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &leaderboardUpdateManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}
