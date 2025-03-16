package round

// app/round/module.go

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	roundrsvp "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/round_rsvp"
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// InitializeRoundModule initializes the Round domain module.
func InitializeRoundModule(
	ctx context.Context,
	session discord.Session,
	interactionRegistry *interactions.Registry,
	publisher eventbus.EventBus,
	logger observability.Logger,
	config *config.Config,
	eventUtil utils.EventUtil,
	helper utils.Helpers,
	interactionStore storage.ISInterface,
) error {
	// Initialize Discord services
	roundDiscord, err := rounddiscord.NewRoundDiscord(ctx, session, publisher, logger, helper, config, interactionStore)
	if err != nil {
		logger.Error(ctx, "Failed to initialize user Discord services", attr.Error(err))
		return err
	}

	// Register Discord interactions
	createround.RegisterHandlers(interactionRegistry, roundDiscord.GetCreateRoundManager())
	roundrsvp.RegisterHandlers(interactionRegistry, roundDiscord.GetRoundRsvpManager())

	// Initialize Watermill handlers (no need to register with router here)
	roundhandlers.NewRoundHandlers(logger, config, eventUtil, helper, roundDiscord)
	return nil
}
