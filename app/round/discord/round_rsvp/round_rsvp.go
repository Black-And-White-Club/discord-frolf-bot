package roundrsvp

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// RoundRsvpManager defines the interface for create round operations.
type RoundRsvpManager interface {
	HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate)
	UpdateRoundEventEmbed(channelID string, messageID roundtypes.EventMessageID, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.Participant) error
	InteractionJoinRoundLate(ctx context.Context, i *discordgo.InteractionCreate)
}

// RoundRsvpManager implements the RoundRsvpManager interface.
type roundRsvpManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
}

// NewRoundRsvpManager creates a new RoundRsvpManager instance.
func NewRoundRsvpManager(session discord.Session, publisher eventbus.EventBus, logger *slog.Logger, helper utils.Helpers, config *config.Config, interactionStore storage.ISInterface) RoundRsvpManager {
	logger.Info(context.Background(), "Creating RoundRsvpManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &roundRsvpManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
	}
}
