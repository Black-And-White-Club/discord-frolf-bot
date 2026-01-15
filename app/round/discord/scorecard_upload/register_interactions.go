package scorecardupload

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, messageRegistry *interactions.MessageRegistry, manager ScorecardUploadManager) {
	// Scorecard upload button (from round embeds)
	registry.RegisterHandler("round_upload_scorecard|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling scorecard upload button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
		)
		manager.HandleScorecardUploadButton(ctx, i)
	})

	// Scorecard upload modal submission
	registry.RegisterHandler("scorecard_upload_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.InfoContext(ctx, "Handling scorecard upload modal submission",
			attr.String("interaction_id", i.ID),
			attr.String("user_id", i.Member.User.ID),
		)
		manager.HandleScorecardUploadModalSubmit(ctx, i)
	})

	// File upload message listener - adapter to provide context to legacy handler
	messageRegistry.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		// manager.HandleFileUploadMessage expects (discord.Session, *discordgo.MessageCreate)
		manager.HandleFileUploadMessage(s, m)
	})
}
