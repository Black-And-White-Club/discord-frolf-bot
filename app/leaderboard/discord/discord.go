package leaderboarddiscord

import (
	"context"
	"log/slog"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	claimtag "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/claim_tag"
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
	GetClaimTagManager() claimtag.ClaimTagManager
}

// LeaderboardDiscord encapsulates all leaderboard-related Discord services.
type LeaderboardDiscord struct {
	LeaderboardUpdateManager leaderboardupdated.LeaderboardUpdateManager
	ClaimTagManager          claimtag.ClaimTagManager
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

	claimTagManager := claimtag.NewClaimTagManager(session, publisher, logger, helper, config, interactionStore, tracer, metrics)

	return &LeaderboardDiscord{
		LeaderboardUpdateManager: leaderboardUpdateManager,
		ClaimTagManager:          claimTagManager,
	}, nil
}

// GetLeaderboardUpdateManager returns the LeaderboardUpdateManager.
func (ld *LeaderboardDiscord) GetLeaderboardUpdateManager() leaderboardupdated.LeaderboardUpdateManager {
	return ld.LeaderboardUpdateManager
}

// GetClaimTagManager returns the ClaimTagManager.
func (ld *LeaderboardDiscord) GetClaimTagManager() claimtag.ClaimTagManager {
	return ld.ClaimTagManager
}
