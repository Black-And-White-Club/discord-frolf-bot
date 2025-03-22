package deleteround

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers interaction handlers for deleting rounds.
func RegisterHandlers(registry *interactions.Registry, manager DeleteRoundManager) {
	if manager == nil {
		slog.Error("❌ DeleteRoundManager is nil! Handlers will not work.")
		return
	}

	slog.Info("RegisterHandlers is registering handlers for DeleteRoundManager")

	registry.RegisterHandler("round_delete", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		slog.Info("Checking delete button interaction", attr.String("custom_id", customID))

		// ✅ Ensure it matches "round_delete|<id>"
		if !strings.HasPrefix(customID, "round_delete|") {
			slog.Warn("❌ Unexpected button interaction", attr.String("custom_id", customID))
			return
		}

		slog.Info("✅ Button matched! Processing delete request.")
		manager.HandleDeleteRound(ctx, i)
	})

}
