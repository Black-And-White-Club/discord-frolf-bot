package udisc

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

// RegisterUDiscInteractions registers all UDisc-related interaction handlers.
func RegisterUDiscInteractions(
	registry *interactions.Registry,
	manager UDiscManager,
) {
	// Register slash command handler
	registry.RegisterHandler("set-udisc-name", func(ctx context.Context, i *discordgo.InteractionCreate) {
		_, _ = manager.HandleSetUDiscNameCommand(ctx, i)
	})
}
