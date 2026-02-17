package updateround

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers interaction handlers for updating rounds.
func RegisterHandlers(registry *interactions.Registry, manager UpdateRoundManager) {
	if manager == nil {
		slog.Error("UpdateRoundManager is nil! Handlers will not work.")
		return
	}

	slog.Info("RegisterHandlers is registering handlers for UpdateRoundManager")

	// Register Edit button handler
	registry.RegisterMutatingHandler("round_edit|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		slog.Info("Edit button interaction received", attr.String("custom_id", customID))

		result, err := manager.HandleEditRoundButton(ctx, i)
		slog.Info("HandleEditRoundButton completed",
			attr.Any("result", result),
			attr.Error(err))
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.EditorRequired, RequiresSetup: true})

	// Register Modal submission handler (THIS WAS MISSING!)
	registry.RegisterMutatingHandler("update_round_modal|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.ModalSubmitData().CustomID
		slog.Info("Update round modal submission received", attr.String("custom_id", customID))

		result, err := manager.HandleUpdateRoundModalSubmit(ctx, i)
		slog.Info("HandleUpdateRoundModalSubmit completed",
			attr.Any("result", result),
			attr.Error(err))
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.EditorRequired, RequiresSetup: true})

	// Register Modal cancel handler (if you have one)
	registry.RegisterMutatingHandler("update_round_modal_cancel|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		slog.Info("Update round modal cancel received", attr.String("custom_id", customID))

		result, err := manager.HandleUpdateRoundModalCancel(ctx, i)
		slog.Info("HandleUpdateRoundModalCancel completed",
			attr.Any("result", result),
			attr.Error(err))
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.EditorRequired, RequiresSetup: true})
}
