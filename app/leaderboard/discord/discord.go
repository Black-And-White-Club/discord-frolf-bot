package leaderboarddiscord

import (
	"context"
	"log/slog"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/trace"
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
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (LeaderboardDiscordInterface, error) {
	leaderboardUpdateManager := leaderboardupdated.NewLeaderboardUpdateManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics)

	return &LeaderboardDiscord{
		LeaderboardUpdateManager: leaderboardUpdateManager,
	}, nil
}

// GetLeaderboardUpdateManager returns the LeaderboardUpdateManager.
func (ld *LeaderboardDiscord) GetLeaderboardUpdateManager() leaderboardupdated.LeaderboardUpdateManager {
	return ld.LeaderboardUpdateManager
}
