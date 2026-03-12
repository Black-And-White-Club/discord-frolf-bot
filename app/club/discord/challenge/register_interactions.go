package challenge

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// RegisterHandlers registers challenge slash commands and button interactions.
func RegisterHandlers(registry *interactions.Registry, manager Manager) {
	registry.RegisterMutatingHandler("challenge", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /challenge command",
			attr.String("guild_id", i.GuildID),
			attr.String("user_id", interactionUserIDForLog(i)),
		)
		if err := manager.HandleChallengeCommand(ctx, i); err != nil {
			slog.Error("Challenge command failed", attr.Error(err))
		}
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler(challengeAcceptPrefix, func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling challenge accept button",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("user_id", interactionUserIDForLog(i)),
		)
		if err := manager.HandleAcceptButton(ctx, i); err != nil {
			slog.Error("Challenge accept failed", attr.Error(err))
		}
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler(challengeDeclinePrefix, func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling challenge decline button",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("user_id", interactionUserIDForLog(i)),
		)
		if err := manager.HandleDeclineButton(ctx, i); err != nil {
			slog.Error("Challenge decline failed", attr.Error(err))
		}
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	registry.RegisterMutatingHandler(challengeSchedulePrefix, func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling challenge schedule button",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("user_id", interactionUserIDForLog(i)),
		)
		if err := manager.HandleScheduleButton(ctx, i); err != nil {
			slog.Error("Challenge schedule failed", attr.Error(err))
		}
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})
}

func interactionUserIDForLog(i *discordgo.InteractionCreate) string {
	switch {
	case i != nil && i.Member != nil && i.Member.User != nil:
		return i.Member.User.ID
	case i != nil && i.User != nil:
		return i.User.ID
	default:
		return ""
	}
}
