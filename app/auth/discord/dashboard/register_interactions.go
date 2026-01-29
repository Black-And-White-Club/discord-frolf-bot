package dashboard

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers dashboard interaction handlers
func RegisterHandlers(registry *interactions.Registry, manager DashboardManager) {
	// /dashboard requires player role and guild setup
	registry.RegisterHandlerWithPermissions("dashboard", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /dashboard command",
			attr.String("user_id", i.Member.User.ID),
			attr.String("guild_id", i.GuildID),
		)
		if err := manager.HandleDashboardCommand(ctx, i); err != nil {
			slog.Error("Failed to handle dashboard command", attr.Error(err))
		}
	}, interactions.PlayerRequired, true)
}
