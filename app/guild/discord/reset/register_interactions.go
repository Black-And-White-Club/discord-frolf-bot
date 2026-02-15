package reset

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers reset-related interaction handlers.
func RegisterHandlers(registry *interactions.Registry, manager ResetManager) {
	// frolf-reset command requires Discord Admin permissions (checked by Discord)
	registry.RegisterMutatingHandler("frolf-reset", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetCommand(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.NoPermissionRequired, RequiresSetup: false}) // Discord admin perms handled by Discord itself

	// Confirmation button
	registry.RegisterMutatingHandler("frolf_reset_confirm", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetConfirmButton(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.NoPermissionRequired, RequiresSetup: false})

	// Cancel button
	registry.RegisterMutatingHandler("frolf_reset_cancel", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_ = manager.HandleResetCancelButton(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.NoPermissionRequired, RequiresSetup: false})
}
