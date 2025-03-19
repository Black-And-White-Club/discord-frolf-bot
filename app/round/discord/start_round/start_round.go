package startround

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// StartRoundManager defines the interface for create round operations.
type StartRoundManager interface {
	TransformRoundToScorecard(payload *roundevents.DiscordRoundStartPayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error)
	UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayload) error
}

// startRoundManager implements the StartRoundManager interface.
type startRoundManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewStartRoundManager creates a new StartRoundManager instance.
func NewStartRoundManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config) StartRoundManager {
	logger.Info(context.Background(), "Creating StartRoundManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &startRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}
