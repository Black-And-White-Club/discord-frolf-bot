package roundrsvp

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager RoundRsvpManager) {
	registry.RegisterMutatingHandler("round_accept|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_accept button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleRoundResponse(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler("round_decline|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_decline button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleRoundResponse(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler("round_tentative|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_tentative button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleRoundResponse(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler("round_join_late|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_join_late button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.InteractionJoinRoundLate(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})
}
