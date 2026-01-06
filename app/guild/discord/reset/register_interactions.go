package reset

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers reset-related interaction handlers.
func RegisterHandlers(registry *interactions.Registry, manager ResetManager) {
	// frolf-reset command requires Discord Admin permissions (checked by Discord)
	registry.RegisterHandlerWithPermissions("frolf-reset", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /frolf-reset command", attr.String("command_name", i.ApplicationCommandData().Name))
		if err := manager.HandleResetCommand(ctx, i); err != nil {
			slog.Error("Failed to handle frolf-reset command", attr.Error(err))
		}
	}, interactions.NoPermissionRequired, false) // Discord admin perms handled by Discord itself

	// Confirmation button
	registry.RegisterHandlerWithPermissions("frolf_reset_confirm", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling reset confirmation button", attr.String("custom_id", i.MessageComponentData().CustomID))
		if err := manager.HandleResetConfirmButton(ctx, i); err != nil {
			slog.Error("Failed to handle reset confirmation button", attr.Error(err))
		}
	}, interactions.NoPermissionRequired, false)

	// Cancel button
	registry.RegisterHandlerWithPermissions("frolf_reset_cancel", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling reset cancel button", attr.String("custom_id", i.MessageComponentData().CustomID))
		if err := manager.HandleResetCancelButton(ctx, i); err != nil {
			slog.Error("Failed to handle reset cancel button", attr.Error(err))
		}
	}, interactions.NoPermissionRequired, false)
}
