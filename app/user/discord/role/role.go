package role

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

// RoleManager defines the interface for create round operations.
type RoleManager interface {
	AddRoleToUser(ctx context.Context, guildID, userID, roleID string) error
	EditRoleUpdateResponse(ctx context.Context, correlationID string, content string) error
	HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate)
	RespondToRoleRequest(ctx context.Context, interactionID, interactionToken, targetUserID string) error
	RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken, requesterID, selectedRole, targetUserID string) error
}

// roleManager implements the RoleManager interface.
type roleManager struct {
	session          discord.Session
	operations       discord.Operations
	publisher        eventbus.EventBus
	logger           observability.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
}

// NewCreateRoleManager creates a new CreateRoleManager instance.
func NewRoleManager(session discord.Session, operations discord.Operations, publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, interactionStore storage.ISInterface) RoleManager {
	logger.Info(context.Background(), "Creating RoleManager",
		attr.Any("session", session),
		attr.Any("operations", operations),
		attr.Any("publisher", publisher),
		attr.Any("config", config),
	)
	return &roleManager{
		session:          session,
		operations:       operations,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
	}
}

// createEvent is a helper function to create a Watermill message.
func (rm *roleManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	// Ensure Metadata is initialized
	if newEvent.Metadata == nil {
		newEvent.Metadata = make(map[string]string)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		rm.logger.Error(ctx, "Failed to marshal payload in CreateResultMessage", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	newEvent.Payload = payloadBytes

	// Set metadata fields
	newEvent.Metadata.Set("handler_name", "Create Original Discord Message to Backend")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", rm.config.Discord.GuildID)

	return newEvent, nil
}
