package roundhandlers

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for round handlers.
type Handlers interface {
	HandleRoundStarted(msg *message.Message) ([]*message.Message, error)
	HandleRoundScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundReminder(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error)
	HandleRoundFinalized(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleted(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreated(msg *message.Message) ([]*message.Message, error)
}

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	Logger    observability.Logger
	Session   discord.Session
	Config    *config.Config
	EventUtil utils.EventUtil
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(logger observability.Logger, session discord.Session, config *config.Config, eventUtil utils.EventUtil) Handlers {
	return &RoundHandlers{
		Logger:    logger,
		Session:   session,
		Config:    config,
		EventUtil: eventUtil,
	}
}
