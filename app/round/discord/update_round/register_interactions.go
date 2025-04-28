package updateround

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// RegisterHandlers registers interaction handlers for updating rounds.
func RegisterHandlers(registry *interactions.Registry, manager UpdateRoundManager) {
	if manager == nil {
		slog.Error("UpdateRoundManager is nil! Handlers will not work.")
		return
	}

	slog.Info("RegisterHandlers is registering handlers for UpdateRoundManager")

	registry.RegisterHandler("round_edit", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		slog.Info("Checking edit button interaction", attr.String("custom_id", customID))

		if !strings.HasPrefix(customID, "round_edit|") {
			slog.Warn("Unexpected button interaction", attr.String("custom_id", customID))
			return
		}

		// Extract the UUID string
		roundIDStr := strings.TrimPrefix(customID, "round_edit|")

		// Parse the UUID properly
		roundUUID, err := uuid.Parse(roundIDStr)
		if err != nil {
			slog.Error("Invalid Round ID format",
				attr.String("custom_id", customID),
				attr.String("round_id_string", roundIDStr),
				attr.Error(err))
			return
		}

		slog.Info("Button matched! Processing edit request.", attr.String("round_id", roundUUID.String()))

		manager.HandleEditRoundButton(ctx, i)
	})
}
