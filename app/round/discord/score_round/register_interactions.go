package scoreround

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager ScoreRoundManager) {
	// Standard score entry button
	registry.RegisterHandler("round_enter_score|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling round_enter_score button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreButton(ctx, i)
	})

	// Unified override button (finalized only, handles single or multi-line overrides)
	registry.RegisterHandler("round_bulk_score_override|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling round_bulk_score_override button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreButton(ctx, i) // Reuse handler; it will detect finalized & permissions; we'll branch in HandleScoreButton for bulk
	})

	// Standard modal submission
	registry.RegisterHandler("submit_score_modal|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling score modal submission",
			attr.String("custom_id", i.ModalSubmitData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreSubmission(ctx, i)
	})

	// Bulk override modal submission (finalized rounds, supports single or multiple lines)
	registry.RegisterHandler("submit_score_bulk_override|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling bulk score override submission",
			attr.String("custom_id", i.ModalSubmitData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("user_username", i.Member.User.Username),
		)
		manager.HandleScoreSubmission(ctx, i)
	})

	// Deprecated handlers (submit_score_modal_finalized| / submit_score_override|) intentionally not registered anymore.
}
