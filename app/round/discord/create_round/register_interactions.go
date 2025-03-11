package createround

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager CreateRoundManager) {
	registry.RegisterHandler("createround", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /createround command", attr.String("command_name", i.ApplicationCommandData().Name))
		manager.HandleCreateRoundCommand(ctx, i)
	})

	registry.RegisterHandler("create_round_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling create_round_modal submission", attr.String("custom_id", i.ModalSubmitData().CustomID))
		manager.HandleCreateRoundModalSubmit(ctx, i)
	})

	registry.RegisterHandler("retry_create_round", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling retry_create_round button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRetryCreateRound(ctx, i)
	})

	registry.RegisterHandler("round_accept|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_accept button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoundResponse(ctx, i, "accepted")
	})

	registry.RegisterHandler("round_decline|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_decline button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoundResponse(ctx, i, "declined")
	})

	registry.RegisterHandler("round_tentative|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling round_tentative button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoundResponse(ctx, i, "tentative")
	})
}
