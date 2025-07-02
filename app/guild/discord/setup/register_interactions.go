package setup

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager SetupManager) {
	registry.RegisterHandler("frolf-setup", func(ctx context.Context, i *discordgo.InteractionCreate) {
		if err := manager.HandleSetupCommand(ctx, i); err != nil {
			// Error logging is handled in the manager
		}
	})
}
