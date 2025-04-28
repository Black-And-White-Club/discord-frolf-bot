package deleteround

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
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

		// ✅ Ensure it matches "round_delete|<uuid>"
		if !strings.HasPrefix(customID, "round_delete|") {
			slog.Warn("❌ Unexpected button interaction", attr.String("custom_id", customID))
			return
		}

		// Extract the UUID string
		roundIDStr := strings.TrimPrefix(customID, "round_delete|")

		// Parse the UUID
		roundUUID, err := uuid.Parse(roundIDStr)
		if err != nil {
			slog.Error("❌ Invalid Round ID format",
				attr.String("custom_id", customID),
				attr.String("round_id_string", roundIDStr),
				attr.Error(err))
			return
		}

		slog.Info("✅ Button matched! Processing delete request.", attr.String("round_id", roundUUID.String()))
		manager.HandleDeleteRoundButton(ctx, i)
	})
}
