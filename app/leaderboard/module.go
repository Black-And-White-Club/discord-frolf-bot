package leaderboard

// app/leaderboard/module.go

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	leaderboardupdated "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord/leaderboard_updated"
	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// InitializeLeaderboardModule initializes the leaderboard domain module.
func InitializeLeaderboardModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
) error {
	// Initialize Discord services
	leaderboardDiscord, err := leaderboarddiscord.NewLeaderboardDiscord(ctx, session, publisher, logger, helper, config)
	if err != nil {
		logger.Error(ctx, "Failed to initialize leaderboard Discord services", attr.Error(err))
		return err
	}

	// Register Discord interactions
	leaderboardupdated.RegisterHandlers(interactionRegistry, leaderboardDiscord.GetLeaderboardUpdateManager())

	// Initialize Watermill handlers (no need to register with router here)
	leaderboardhandlers.NewLeaderboardHandlers(logger, config, eventUtil, helper, leaderboardDiscord)
	return nil
}
