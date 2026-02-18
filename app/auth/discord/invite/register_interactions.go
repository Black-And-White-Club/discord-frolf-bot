package invite

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers the /invite slash command handler.
func RegisterHandlers(registry *interactions.Registry, manager InviteManager) {
	registry.RegisterHandlerWithPermissions("invite", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /invite command",
			attr.String("user_id", i.Member.User.ID),
			attr.String("guild_id", i.GuildID),
		)
		manager.HandleInviteCommand(ctx, i)
	}, interactions.EditorRequired, true)
}
