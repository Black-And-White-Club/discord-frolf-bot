package leaderboardupdated

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager LeaderboardUpdateManager) {
	// Button handler for leaderboard pagination
	registry.RegisterHandler("leaderboard_prev|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling leaderboard previous button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleLeaderboardPagination(ctx, i)
	})

	registry.RegisterHandler("leaderboard_next|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling leaderboard next button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleLeaderboardPagination(ctx, i)
	})
}
