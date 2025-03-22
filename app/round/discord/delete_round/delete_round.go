package deleteround

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// DeleteRoundManager defines the interface for create round operations.
type DeleteRoundManager interface {
	DeleteEmbed(ctx context.Context, eventMessageID roundtypes.EventMessageID, channelID string) (bool, error)
	HandleDeleteRound(ctx context.Context, i *discordgo.InteractionCreate)
}

// deleteRoundManager implements the DeleteRoundManager interface.
type deleteRoundManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewDeleteRoundManager creates a new DeleteRoundManager instance.
func NewDeleteRoundManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config) DeleteRoundManager {
	logger.Info(context.Background(), "Creating DeleteRoundManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &deleteRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}
