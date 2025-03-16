package roundreminder

import (
	"context"
	"encoding/json"
	"fmt"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// RoundReminderManager defines the interface for create round operations.
type RoundReminderManager interface {
}

// RoundReminderManager implements the RoundReminderManager interface.
type roundReminderManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           observability.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
}

// NewRoundReminderManager creates a new RoundReminderManager instance.
func NewRoundReminderManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, interactionStore storage.ISInterface) RoundReminderManager {
	logger.Info(context.Background(), "Creating RoundReminderManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &roundReminderManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
	}
}

// createEvent is a helper function to create a Watermill message.
func (rrm *roundReminderManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		rrm.logger.Error(ctx, "Failed to marshal payload in CreateResultMessage", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w, original error: %v", err, err)
	}
	newEvent.Payload = payloadBytes
	newEvent.Metadata.Set("handler_name", "Create Original Discord Message to Backend")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", rrm.config.Discord.GuildID)
	return newEvent, nil
}
