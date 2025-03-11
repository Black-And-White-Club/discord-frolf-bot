package roundhandlers

import (
	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
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
	HandleRoundValidationFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreationFailed(msg *message.Message) ([]*message.Message, error)
}

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	Logger       observability.Logger
	Config       *config.Config
	EventUtil    utils.EventUtil
	Helpers      utils.Helpers
	RoundDiscord rounddiscord.RoundDiscordInterface
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(logger observability.Logger, config *config.Config, eventUtil utils.EventUtil, helpers utils.Helpers, roundDiscord rounddiscord.RoundDiscordInterface) Handlers {
	return &RoundHandlers{
		Logger:       logger,
		Config:       config,
		EventUtil:    eventUtil,
		Helpers:      helpers,
		RoundDiscord: roundDiscord,
	}
}
