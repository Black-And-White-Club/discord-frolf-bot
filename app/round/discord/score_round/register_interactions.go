package scoreround

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager ScoreRoundManager) {
	// Button for entering a score
	registry.RegisterHandler("enter_score|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling enter_score button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleScoreButton(ctx, i)
	})

	// Modal submission for score input
	registry.RegisterHandler("submit_score_modal|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling score modal submission",
			attr.String("custom_id", i.ModalSubmitData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user", i.Member.User.Username))
		manager.HandleScoreSubmission(ctx, i)
	})
}
