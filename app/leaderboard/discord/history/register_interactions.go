package history

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers the history command handlers.
func RegisterHandlers(registry *interactions.Registry, manager HistoryManager) {
	registry.RegisterHandler("history", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling history command",
			attr.String("interaction_id", i.ID))
		manager.HandleHistoryCommand(ctx, i)
	})
}
