package userhandlers

import (
	"log/slog"
	"sync"

	cache "github.com/Black-And-White-Club/discord-frolf-bot/bigcache"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	userevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/helpers"
	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for user handlers.
type Handlers interface {
	HandleReaction(s discord.Discord, r *discord.MessageReactionAdd)
	HandleMessageCreate(s discord.Discord, m *discord.MessageCreate)
	HandleIncludeTagNumberRequest(s discord.Discord, m *discord.MessageCreate, wm *message.Message)
	HandleIncludeTagNumberResponse(s discord.Discord, m *discord.MessageCreate, wm *message.Message)
	HandleRoleUpdateCommand(discord.Discord, *discord.MessageCreate, *message.Message)
	HandleRoleResponse(s discord.Discord, m *discord.MessageCreate, wm *message.Message)
	HandleRoleUpdateResponse(s discord.Discord, msg *message.Message) error
	HandleAskIfUserHasTag(s discord.Discord, wm *message.Message)
	HandleSignupRequest(s discord.Discord, m *discord.MessageCreate, wm *message.Message)
	HandleSignupResponse(msg *message.Message) error
}

// UserHandlers handles user-related events.
type UserHandlers struct {
	Logger         *slog.Logger
	EventBus       eventbus.EventBus
	Session        discord.Discord
	Config         *config.Config
	UserChannelMap map[string]string
	UserChannelMu  *sync.RWMutex
	Cache          cache.CacheInterface
	EventUtil      utils.EventUtil
	ErrorReporter  errors.ErrorReporterInterface
}

// NewUserHandlers creates a new UserHandlers.
func NewUserHandlers(logger *slog.Logger, eventBus eventbus.EventBus, session discord.Discord, config *config.Config, cache cache.CacheInterface, eventUtil utils.EventUtil, errorReport errors.ErrorReporterInterface) Handlers {
	return &UserHandlers{
		Logger:         logger,
		EventBus:       eventBus,
		Session:        session,
		Config:         config,
		UserChannelMap: make(map[string]string),
		UserChannelMu:  &sync.RWMutex{},
		Cache:          cache,
		EventUtil:      eventUtil,
		ErrorReporter:  errorReport,
	}
}

func (h *UserHandlers) getChannelID(s discord.Discord, userID string) (string, error) {
	h.UserChannelMu.RLock()
	cachedID, ok := h.UserChannelMap[userID]
	h.UserChannelMu.RUnlock()
	if ok {
		return cachedID, nil
	}

	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		return "", err
	}

	h.UserChannelMu.Lock()
	h.UserChannelMap[userID] = channel.ID
	h.UserChannelMu.Unlock()

	return channel.ID, nil
}

// publishCancelEvent publishes a cancel event.
func (h *UserHandlers) publishCancelEvent(correlationID, userID string) {
	payload := userevents.CancelPayload{
		UserID: userID,
	}
	helpers.PublishEvent(h.EventBus, userevents.SignupCanceled, correlationID, payload)
}

// publishTraceEvent publishes a trace event.
func (h *UserHandlers) publishTraceEvent(correlationID, msg string) { // Need this to be able to take in trace event as argument
	tracePayload := userevents.TracePayload{
		Message: msg,
	}
	helpers.PublishEvent(h.EventBus, userevents.SignupTrace, correlationID, tracePayload)
}

// sendUserMessage sends a message to the user.
func (h *UserHandlers) sendUserMessage(s discord.Discord, userID, messageText, correlationID string) {
	// Wrapper function to match the expected signature
	errorReporter := func(correlationID string, msg string, err error) {
		h.ErrorReporter.ReportError(correlationID, msg, err)
	}

	helpers.SendUserMessage(s, userID, messageText, correlationID, h.getChannelID, errorReporter)
}
