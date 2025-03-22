package leaderboarddiscord

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// LeaderboardDiscordInterface defines the interface for LeaderboardDiscord.
type LeaderboardDiscordInterface interface {
	GetLeaderboardUpdateManager() leaderboardupdated.LeaderboardUpdateManager
}

// LeaderboardDiscord encapsulates all leaderboard-related Discord services.
type LeaderboardDiscord struct {
	LeaderboardUpdateManager leaderboardupdated.LeaderboardUpdateManager
}

// NewLeaderboardDiscord creates a new LeaderboardDiscord instance.
func NewLeaderboardDiscord(
	ctx context.Context,
	session discord.Session,
	publisher eventbus.EventBus,
	logger observability.Logger,
	helper utils.Helpers,
	config *config.Config,
) (LeaderboardDiscordInterface, error) {
	leaderboardUpdateManager := leaderboardupdated.NewCreateLeaderboardUpdateManager(session, publisher, logger, helper, config)

	return &LeaderboardDiscord{
		LeaderboardUpdateManager: leaderboardUpdateManager,
	}, nil
}

// GetLeaderboardUpdateManager returns the LeaderboardUpdateManager.
func (ld *LeaderboardDiscord) GetLeaderboardUpdateManager() leaderboardupdated.LeaderboardUpdateManager {
	return ld.LeaderboardUpdateManager
}
