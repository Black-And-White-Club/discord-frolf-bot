// app/user/module.go
package user

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// InitializeUserModule initializes the user domain module.
func InitializeUserModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	reactionRegistry *interactions.ReactionRegistry,
	publisher eventbus.EventBus,
	logger observability.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
) error {
	// Initialize Discord services
	userDiscord, err := userdiscord.NewUserDiscord(ctx, session, publisher, logger, helper, config, interactionStore)
	if err != nil {
		logger.Error(ctx, "Failed to initialize user Discord services", attr.Error(err))
		return err
	}

	// Register Discord interactions
	role.RegisterHandlers(interactionRegistry, userDiscord.GetRoleManager())
	signup.RegisterHandlers(interactionRegistry, userDiscord.GetSignupManager())

	// Initialize Watermill handlers (no need to register with router here)
	userhandlers.NewUserHandlers(logger, config, eventUtil, helper, userDiscord)

	// Register reaction handlers
	reactionRegistry.RegisterMessageReactionAddHandler(userDiscord.GetSignupManager().MessageReactionAdd)

	return nil
}
