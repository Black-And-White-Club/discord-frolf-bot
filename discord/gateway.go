// discord/gateway.go
package discord

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GatewayEventHandler handles incoming events from the Discord Gateway.
type GatewayEventHandler interface {
	RegisterHandlers()
}

type gatewayEventHandler struct {
	publisher eventbus.EventBus // Use your eventbus interface
	logger    observability.Logger
	helper    utils.Helpers
	config    *config.Config
	session   Session
	discord   Operations
}

// NewGatewayEventHandler creates a new GatewayEventHandler.
func NewGatewayEventHandler(publisher eventbus.EventBus, logger observability.Logger, helper utils.Helpers, config *config.Config, session Session, discord Operations) GatewayEventHandler {
	return &gatewayEventHandler{
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		session:   session,
		discord:   discord,
	}
}

// RegisterHandlers registers all the Discord gateway event handlers.
func (h *gatewayEventHandler) RegisterHandlers() {
	h.session.AddHandler(h.interactionCreate)
	h.session.AddHandler(h.messageReactionAdd)

}

// createEvent is a helper function to create a Watermill message.
func (h *gatewayEventHandler) createEvent(_ context.Context, topic string, payload interface{}) (*message.Message, error) {
	msg, err := h.helper.CreateResultMessage(nil, payload, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	return msg, nil
}
