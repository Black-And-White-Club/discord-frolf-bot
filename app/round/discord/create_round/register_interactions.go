package createround

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager CreateRoundManager) {
	// createround command available to all players
	registry.RegisterMutatingHandler("createround", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /createround command", attr.String("command_name", i.ApplicationCommandData().Name))
		manager.HandleCreateRoundCommand(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	// Modal submissions require same permission as the command
	registry.RegisterMutatingHandler("create_round_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling create_round_modal submission", attr.String("custom_id", i.ModalSubmitData().CustomID))
		manager.HandleCreateRoundModalSubmit(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	// Retry button requires same permission as the command
	registry.RegisterMutatingHandler("retry_create_round", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling retry_create_round button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRetryCreateRound(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})
}
