package finalizeround

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

// FinalizeRoundManager defines the interface for create round operations.
type FinalizeRoundManager interface {
	TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error)
	FinalizeScorecardEmbed(ctx context.Context, eventMessageID, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.Message, error)
}

// finalizeRoundManager implements the FinalizeRoundManager interface.
type finalizeRoundManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewFinalizeRoundManager creates a new FinalizeRoundManager instance.
func NewFinalizeRoundManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config) FinalizeRoundManager {
	logger.Info(context.Background(), "Creating FinalizeRoundManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &finalizeRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}
