package setup

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager SetupManager) {
	registry.RegisterHandler("frolf-setup", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /frolf-setup command", attr.String("command_name", i.ApplicationCommandData().Name))
		if err := manager.HandleSetupCommand(ctx, i); err != nil {
			slog.Error("Failed to handle frolf-setup command", attr.Error(err))
		}
	})

	registry.RegisterHandler("guild_setup_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling guild_setup_modal submission", attr.String("custom_id", i.ModalSubmitData().CustomID))
		if err := manager.HandleSetupModalSubmit(ctx, i); err != nil {
			slog.Error("Failed to handle guild setup modal submission", attr.Error(err))
		}
	})
}
