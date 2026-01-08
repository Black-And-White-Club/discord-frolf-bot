package reset

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers reset-related interaction handlers.
func RegisterHandlers(registry *interactions.Registry, manager ResetManager) {
	// frolf-reset command requires Discord Admin permissions (checked by Discord)
	registry.RegisterHandlerWithPermissions("frolf-reset", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetCommand(ctx, i)
	}, interactions.NoPermissionRequired, false) // Discord admin perms handled by Discord itself

	// Confirmation button
	registry.RegisterHandlerWithPermissions("frolf_reset_confirm", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetConfirmButton(ctx, i)
	}, interactions.NoPermissionRequired, false)

	// Cancel button
	registry.RegisterHandlerWithPermissions("frolf_reset_cancel", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetCancelButton(ctx, i)
	}, interactions.NoPermissionRequired, false)
}
