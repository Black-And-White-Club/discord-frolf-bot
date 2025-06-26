package claimtag

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager ClaimTagManager) {
	registry.RegisterHandler("claimtag", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /claimtag command",
			attr.String("command_name", i.ApplicationCommandData().Name),
			attr.String("user", i.Member.User.Username))

		result, err := manager.HandleClaimTagCommand(ctx, i)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to handle claim tag command",
				attr.Error(err),
				attr.String("user", i.Member.User.Username))
			return
		}

		if result.Error != nil {
			slog.ErrorContext(ctx, "Claim tag command returned error",
				attr.Error(result.Error),
				attr.String("user", i.Member.User.Username))
		} else {
			slog.InfoContext(ctx, "Claim tag command completed successfully",
				attr.String("user", i.Member.User.Username))
		}
	})
}
