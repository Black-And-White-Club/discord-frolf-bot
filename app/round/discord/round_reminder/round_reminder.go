package roundreminder

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

// RoundReminderManager defines the interface for create round operations.
type RoundReminderManager interface {
	SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayload) (bool, error)
}

// RoundReminderManager implements the RoundReminderManager interface.
type roundReminderManager struct {
	session   discord.Session
	publisher eventbus.EventBus
	logger    *slog.Logger
	helper    utils.Helpers
	config    *config.Config
}

// NewRoundReminderManager creates a new RoundReminderManager instance.
func NewRoundReminderManager(session discord.Session, publisher eventbus.EventBus, logger *slog.Logger, helper utils.Helpers, config *config.Config) RoundReminderManager {
	logger.Info(context.Background(), "Creating RoundReminderManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &roundReminderManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
	}
}
