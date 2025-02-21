package roundhandlers

import (
	"log/slog"

	cache "github.com/Black-And-White-Club/discord-frolf-bot/bigcache"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/events/round"
	"github.com/Black-And-White-Club/discord-frolf-bot/helpers"
	roundservice "github.com/Black-And-White-Club/discord-frolf-bot/services/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for round handlers.
type Handlers interface {
	HandleCreateRoundCommand(m *discord.MessageCreate, wm *message.Message)
	SendReminder(msg *message.Message)
	HandleRoundCreated(msg *message.Message) error
	HandleRoundDeleted(msg *message.Message) error
	HandleRoundUpdated(msg *message.Message) error
	HandleRoundTitleCollected(msg *message.Message) error
	HandleRoundTitleResponse(msg *message.Message) error
	HandleRoundStartTimeCollected(msg *message.Message) error
	HandleRoundStartTimeResponse(msg *message.Message) error
	HandleRoundLocationCollected(msg *message.Message) error
	HandleRoundLocationResponse(msg *message.Message) error
	HandleRoundConfirmationRequest(msg *message.Message) error
	HandleRoundConfirmationResponse(msg *message.Message) error
	HandleRoundEmbedEvent(msg *message.Message) error
}

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	logger          *slog.Logger
	EventBus        eventbus.EventBus
	Session         discord.Session
	Config          *config.Config
	Cache           cache.CacheInterface
	EventUtil       utils.EventUtil
	ErrorReporter   errors.ErrorReporterInterface
	ChannelIDGetter helpers.ChannelIDGetter
	EmbedService    roundservice.EmbedServiceInterface
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(logger *slog.Logger, eventBus eventbus.EventBus, session discord.Session, config *config.Config, cache cache.CacheInterface, eventUtil utils.EventUtil, errorReporter errors.ErrorReporterInterface, channelIDGetter helpers.ChannelIDGetter, embedService roundservice.EmbedServiceInterface) Handlers {
	return &RoundHandlers{
		logger:          logger,
		EventBus:        eventBus,
		Session:         session,
		Config:          config,
		Cache:           cache,
		EventUtil:       eventUtil,
		ErrorReporter:   errorReporter,
		ChannelIDGetter: channelIDGetter,
		EmbedService:    embedService,
	}
}

// publishCancelEvent publishes a cancel event.
func (h *RoundHandlers) publishCancelEvent(correlationID, userID string) {
	payload := discordroundevents.CancelRoundCreationPayload{
		UserID: userID,
	}
	helpers.PublishEvent(h.EventBus, discordroundevents.RoundCreationCanceled, correlationID, payload, h.EventUtil, h.ErrorReporter)
}

// publishTraceEvent publishes a trace event.
func (h *RoundHandlers) publishTraceEvent(correlationID, event string, msg string) {
	tracePayload := discordroundevents.TracePayload{
		Message: msg,
	}
	helpers.PublishEvent(h.EventBus, event, correlationID, tracePayload, h.EventUtil, h.ErrorReporter)
}

// sendUserMessage sends a message to the user using the helpers package.
func (h *RoundHandlers) sendUserMessage(userID, messageText, correlationID string) {
	err := helpers.SendUserMessage(h.Session, h.ChannelIDGetter, userID, messageText, h.ErrorReporter)
	if err != nil {
		h.ErrorReporter.ReportError(correlationID, "Failed to send user message", err, "userID", userID)
	}
}
