package rounddiscord

import (
	"context"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// RoundDiscordInterface defines the interface for RoundDiscord.
type RoundDiscordInterface interface {
	GetCreateRoundManager() createround.CreateRoundManager
}

// RoundDiscord encapsulates all Round Discord services.
type RoundDiscord struct {
	CreateRoundManager createround.CreateRoundManager
}

// NewRoundDiscord creates a new RoundDiscord instance.
func NewRoundDiscord(
	ctx context.Context,
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger observability.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
) (RoundDiscordInterface, error) {
	createRoundManager := createround.NewCreateRoundManager(session, publisher, logger, helper, config, interactionStore)

	return &RoundDiscord{
		CreateRoundManager: createRoundManager,
	}, nil
}

// GetRoleManager returns the RoleManager.
func (rd *RoundDiscord) GetCreateRoundManager() createround.CreateRoundManager {
	return rd.CreateRoundManager
}
