// discord/gateway.go
package discord

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// GatewayEventHandler handles incoming events from the Discord Gateway.
type GatewayEventHandler interface {
	MessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd)
	HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd)
	InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate)
	HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate)
	HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate)
	HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate)
	HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate)
}

type gatewayEventHandler struct {
	publisher  eventbus.EventBus
	logger     observability.Logger
	helper     utils.Helpers
	config     *config.Config
	session    Session
	discord    Operations
	tokenStore *storage.TokenStore
}

// NewGatewayEventHandler creates a new GatewayEventHandler.
func NewGatewayEventHandler(publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, session Session, discord Operations) GatewayEventHandler {
	logger.Info(context.Background(), "Creating GatewayEventHandler",
		attr.Any("publisher", publisher),
		attr.Any("session", session),
		attr.Any("config", config),
		attr.Any("discordOps", discord),
	)
	return &gatewayEventHandler{
		publisher:  publisher,
		logger:     logger,
		helper:     helper,
		config:     config,
		session:    session,
		discord:    discord,
		tokenStore: storage.NewTokenStore(),
	}
}

// createEvent is a helper function to create a Watermill message.
// discord/gateway_interactions.go
func (h *gatewayEventHandler) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) { // Add i *discordgo.InteractionCreate
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error(ctx, "Failed to marshal payload in CreateResultMessage", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	newEvent.Payload = payloadBytes

	newEvent.Metadata.Set("handler_name", "Create Original Discord Message to Backend")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")

	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", h.config.Discord.GuildID)

	return newEvent, nil
}
