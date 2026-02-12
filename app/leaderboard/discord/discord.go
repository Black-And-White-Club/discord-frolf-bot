package leaderboarddiscord

import (
	"context"
	"log/slog"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	claimtag "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/claim_tag"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/history"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/season"
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
	GetSeasonManager() season.SeasonManager
	GetHistoryManager() history.HistoryManager
}

// LeaderboardDiscord encapsulates all leaderboard-related Discord services.
type LeaderboardDiscord struct {
	LeaderboardUpdateManager leaderboardupdated.LeaderboardUpdateManager
	ClaimTagManager          claimtag.ClaimTagManager
	SeasonManager            season.SeasonManager
	HistoryManager           history.HistoryManager
}

// NewLeaderboardDiscord creates a new LeaderboardDiscord instance.

func NewLeaderboardDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (LeaderboardDiscordInterface, error) {
	leaderboardUpdateManager := leaderboardupdated.NewLeaderboardUpdateManager(session, publisher, logger, helper, config, guildConfigResolver, interactionStore, guildConfigCache, tracer, metrics)

	claimTagManager := claimtag.NewClaimTagManager(session, publisher, logger, helper, config, guildConfigResolver, interactionStore, guildConfigCache, tracer, metrics)

	seasonManager := season.NewSeasonManager(session, publisher, logger, helper, config, guildConfigResolver, interactionStore, guildConfigCache, tracer, metrics)

	historyManager := history.NewHistoryManager(session, publisher, logger, helper, interactionStore, metrics)

	return &LeaderboardDiscord{
		LeaderboardUpdateManager: leaderboardUpdateManager,
		ClaimTagManager:          claimTagManager,
		SeasonManager:            seasonManager,
		HistoryManager:           historyManager,
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

// GetSeasonManager returns the SeasonManager.
func (ld *LeaderboardDiscord) GetSeasonManager() season.SeasonManager {
	return ld.SeasonManager
}

// GetHistoryManager returns the HistoryManager.
func (ld *LeaderboardDiscord) GetHistoryManager() history.HistoryManager {
	return ld.HistoryManager
}
