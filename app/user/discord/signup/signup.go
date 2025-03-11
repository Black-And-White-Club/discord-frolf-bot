// signup/signup.go
package signup

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

// SignupManager defines the interface for signup operations.
type SignupManager interface {
	SendSignupModal(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate)
	MessageReactionAdd(s discord.Session, r *discordgo.MessageReactionAdd)
	HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd)
	HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate)
	SendSignupResult(interactionToken string, success bool) error
}

// signupManager implements the SignupManager interface.
type signupManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           observability.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
}

// NewSignupManager creates a new SignupManager instance.
func NewSignupManager(session discord.Session, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, interactionStore storage.ISInterface) (SignupManager, error) {
	logger.Info(context.Background(), "Creating SignupManager",
		attr.Any("session", session),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &signupManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
	}, nil
}

// createEvent is a helper function to create a Watermill message.
func (sm *signupManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	// Ensure Metadata is initialized
	if newEvent.Metadata == nil {
		newEvent.Metadata = make(map[string]string)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		sm.logger.Error(ctx, "Failed to marshal payload in CreateResultMessage", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	newEvent.Payload = payloadBytes

	// Set metadata fields
	newEvent.Metadata.Set("handler_name", "Create Original Discord Message to Backend")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", sm.config.Discord.GuildID)

	return newEvent, nil
}
