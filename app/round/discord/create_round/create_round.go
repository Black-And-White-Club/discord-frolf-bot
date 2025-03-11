package createround

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// CreateRoundManager defines the interface for create round operations.
type CreateRoundManager interface {
	HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate)
	UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) error
	UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) error
	HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate)
	SendRoundEventEmbed(channelID, eventID, title, description string, startTime time.Time, location, creatorID string) (*discordgo.Message, error)
	SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate, response string)
}

// createRoundManager implements the CreateRoundManager interface.
type createRoundManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           observability.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
}

// NewCreateRoundManager creates a new CreateRoundManager instance.
func NewCreateRoundManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, interactionStore storage.ISInterface) CreateRoundManager {
	logger.Info(context.Background(), "Creating CreateRoundManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &createRoundManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
	}
}

// createEvent is a helper function to create a Watermill message.
func (crm *createRoundManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		crm.logger.Error(ctx, "Failed to marshal payload in CreateResultMessage", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w, original error: %v", err, err)
	}
	newEvent.Payload = payloadBytes
	newEvent.Metadata.Set("handler_name", "Create Original Discord Message to Backend")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", crm.config.Discord.GuildID)
	return newEvent, nil
}
