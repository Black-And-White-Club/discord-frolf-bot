package season

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers the season command handlers.
func RegisterHandlers(registry *interactions.Registry, manager SeasonManager) {
	registry.RegisterMutatingHandler("season", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling season command",
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleSeasonCommand(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.AdminRequired, RequiresSetup: true})
}
