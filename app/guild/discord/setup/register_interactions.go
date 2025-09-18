package setup

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager SetupManager) {
	// frolf-setup command requires Discord Admin permissions (checked by Discord, no custom permission needed)
	registry.RegisterHandlerWithPermissions("frolf-setup", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /frolf-setup command", attr.String("command_name", i.ApplicationCommandData().Name))
		if err := manager.HandleSetupCommand(ctx, i); err != nil {
			slog.Error("Failed to handle frolf-setup command", attr.Error(err))
		}
	}, interactions.NoPermissionRequired, false) // Discord admin perms handled by Discord itself

	// Setup modal submission (only accessible to those who can run frolf-setup)
	registry.RegisterHandlerWithPermissions("guild_setup_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling guild_setup_modal submission", attr.String("custom_id", i.ModalSubmitData().CustomID))
		if err := manager.HandleSetupModalSubmit(ctx, i); err != nil {
			slog.Error("Failed to handle guild setup modal submission", attr.Error(err))
		}
	}, interactions.NoPermissionRequired, false)
}
