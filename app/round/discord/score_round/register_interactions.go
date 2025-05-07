package scoreround

import (
	"context"
	"log/slog" // Keep slog import if used here

	// Ensure attr import is present if used in slog calls
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager ScoreRoundManager) {
	registry.RegisterHandler("round_enter_score|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling round_enter_score button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreButton(ctx, i)
	})

	registry.RegisterHandler("submit_score_modal|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling score modal submission",
			attr.String("custom_id", i.ModalSubmitData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreSubmission(ctx, i)
	})
}
