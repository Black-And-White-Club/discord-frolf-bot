package embedpagination

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, session discord.Session) {
	if registry == nil || session == nil {
		return
	}

	registry.RegisterHandler("round_page|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		HandlePageNavigation(ctx, session, i)
	})
}
