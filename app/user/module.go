// app/user/module.go
package user

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel"
)

// InitializeUserModule sets up the user module.
func InitializeUserModule(
	ctx context.Context,
	session discord.Session,
	router *message.Router,
	interactionRegistry *interactions.Registry,
	reactionRegistry *interactions.ReactionRegistry,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
	discordMetrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver, // <-- Add this parameter
) (*userrouter.UserRouter, error) {
	tracer := otel.Tracer("user-module")

	// Initialize Discord services
	userDiscord, err := userdiscord.NewUserDiscord(ctx, session, eventBus, logger, helper, cfg, guildConfigResolver, interactionStore, tracer, discordMetrics)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize user Discord services", attr.Error(err))
		return nil, err
	}

	// Register slash command handlers
	role.RegisterHandlers(interactionRegistry, userDiscord.GetRoleManager())
	signup.RegisterHandlers(interactionRegistry, userDiscord.GetSignupManager())

	// Build Watermill Handlers
	userHandlers := userhandlers.NewUserHandlers(
		logger,
		cfg,
		helper,
		userDiscord,
		tracer,
		discordMetrics,
	)

	// Setup Watermill router
	userRouter := userrouter.NewUserRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helper,
		tracer,
	)
	// Store the userDiscord instance for access to signup manager in other modules
	userRouter.SetUserDiscord(userDiscord)

	if err := userRouter.Configure(ctx, userHandlers); err != nil {
		logger.ErrorContext(ctx, "Failed to configure user router", attr.Error(err))
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	// Register reaction handlers
	reactionRegistry.RegisterMessageReactionAddHandler(func(s discord.Session, r *discordgo.MessageReactionAdd) {
		logger.InfoContext(ctx, "MessageReactionAdd event received",
			attr.String("user_id", r.UserID),
			attr.String("message_id", r.MessageID),
			attr.String("channel_id", r.ChannelID),
			attr.String("guild_id", r.GuildID),
			attr.String("emoji", r.Emoji.Name),
		)

		if _, err := userDiscord.GetSignupManager().MessageReactionAdd(s, r); err != nil {
			logger.ErrorContext(ctx, "Error handling reaction add",
				attr.Error(err),
				attr.String("user_id", r.UserID),
				attr.String("message_id", r.MessageID),
				attr.String("guild_id", r.GuildID),
			)
		}
	})

	return userRouter, nil
}
