// app/user/module.go
package user

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel"
)

// InitializeUserModule initializes the user domain module.
func InitializeUserModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	reactionRegistry *interactions.ReactionRegistry,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	config *config.Config,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
	discordMetricsService discordmetrics.DiscordMetrics, // Inject Discord metrics
) error {
	// Initialize Tracer
	tracer := otel.Tracer("user-module")

	// Initialize Discord services
	userDiscord, err := userdiscord.NewUserDiscord(ctx, session, publisher, logger, helper, config, interactionStore, tracer, discordMetricsService)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to initialize user Discord services", attr.Error(err))
		return err
	}

	// Register Discord interactions
	role.RegisterHandlers(interactionRegistry, userDiscord.GetRoleManager())
	signup.RegisterHandlers(interactionRegistry, userDiscord.GetSignupManager())

	// Initialize Watermill handlers (no need to register with router here)
	userhandlers.NewUserHandlers(logger, config, helper, userDiscord, tracer, discordMetricsService)

	// Register reaction handlers
	reactionRegistry.RegisterMessageReactionAddHandler(func(s discord.Session, r *discordgo.MessageReactionAdd) {
		_, err := userDiscord.GetSignupManager().MessageReactionAdd(s, r)
		if err != nil {
			logger.ErrorContext(ctx, "Error handling reaction add", attr.Error(err), attr.String("user_id", r.UserID), attr.String("message_id", r.MessageID))
			// Consider how you want to handle errors here - potentially log and/or send a message to the user.
		}
	})

	return nil
}
