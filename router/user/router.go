package userrouter

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger   *slog.Logger
	Router   *message.Router
	eventbus eventbus.EventBus
	session  discord.Discord
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(logger *slog.Logger, router *message.Router, eventbus eventbus.EventBus, session discord.Discord) *UserRouter {
	return &UserRouter{
		logger:   logger,
		Router:   router,
		eventbus: eventbus,
		session:  session,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *UserRouter) Configure(handlers userhandlers.Handlers, eventbus eventbus.EventBus) error {
	// Add middleware for logging, retries, and correlation
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		middleware.Retry{
			MaxRetries: 3,
		}.Middleware,
	)

	// Discord-based handlers (require *discord.MessageCreate)
	discordHandlers := map[string]func(discord.Discord, *discord.MessageCreate, *message.Message){
		discorduserevents.RoleUpdateCommand:         handlers.HandleRoleUpdateCommand,
		discorduserevents.RoleSelectRequest:         handlers.HandleRoleResponse,
		discorduserevents.SignupTagAsk:              handlers.HandleSignupRequest,
		discorduserevents.SignupTagIncludeRequested: handlers.HandleIncludeTagNumberRequest,
		discorduserevents.TagNumberRequested:        handlers.HandleIncludeTagNumberResponse,
	}

	// Watermill message-based handlers (only use *message.Message)
	watermillHandlers := map[string]func(*message.Message) error{
		userevents.UserRoleUpdated:      r.wrapHandlerWithSession(handlers.HandleRoleUpdateResponse),
		userevents.UserRoleUpdateFailed: r.wrapHandlerWithSession(handlers.HandleRoleUpdateResponse),
		userevents.UserCreated:          handlers.HandleSignupResponse,
		userevents.UserCreationFailed:   handlers.HandleSignupResponse,
	}

	// Mixed handlers (only use discord.Discord & *message.Message, no *discord.MessageCreate)
	mixedHandlers := map[string]func(discord.Discord, *message.Message){
		discorduserevents.SignupStarted: handlers.HandleAskIfUserHasTag,
	}

	// Register Discord-based handlers
	for topic, handler := range discordHandlers {
		r.addHandler(topic, eventbus, r.wrapDiscordHandler(handler))
	}

	// Register Watermill message-based handlers
	for topic, handler := range watermillHandlers {
		r.addHandler(topic, eventbus, r.wrapMessageHandler(handler))
	}

	// Register mixed handlers
	for topic, handler := range mixedHandlers {
		r.addHandler(topic, eventbus, r.wrapMixedHandler(handler))
	}

	return nil
}

// wrapHandlerWithSession wraps a handler that needs discord.Discord
func (r *UserRouter) wrapHandlerWithSession(handler func(discord.Discord, *message.Message) error) func(*message.Message) error {
	return func(msg *message.Message) error {
		return handler(r.session, msg) // Pass session explicitly
	}
}

// wrapDiscordHandler adapts a Discord-based handler (using *discord.MessageCreate) to match Watermill's message.HandlerFunc.
func (r *UserRouter) wrapDiscordHandler(handler func(discord.Discord, *discord.MessageCreate, *message.Message)) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		var discordMsg discord.MessageCreate
		if err := json.Unmarshal(msg.Payload, &discordMsg); err != nil {
			r.logger.Error("Failed to unmarshal Discord message", slog.String("error", err.Error()))
			return nil, err
		}

		handler(r.session, &discordMsg, msg)
		return nil, nil
	}
}

// wrapMessageHandler adapts a Watermill-only handler (using *message.Message) to match Watermill's message.HandlerFunc.
func (r *UserRouter) wrapMessageHandler(handler func(*message.Message) error) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		if err := handler(msg); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

// wrapMixedHandler adapts handlers that use both `discord.Discord` and `*message.Message`.
func (r *UserRouter) wrapMixedHandler(handler func(discord.Discord, *message.Message)) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		handler(r.session, msg)
		return nil, nil
	}
}

// addHandler registers a handler in the Watermill router.
func (r *UserRouter) addHandler(topic string, eventbus eventbus.EventBus, handler message.HandlerFunc) {
	handlerName := fmt.Sprintf("discord.user.module.handle.%s.%s", topic, watermill.NewUUID())
	r.logger.Info("Registering handler",
		slog.String("topic", topic),
		slog.String("handler", handlerName),
	)

	r.Router.AddHandler(
		handlerName,
		topic,
		eventbus,
		topic,
		eventbus,
		handler,
	)
}

// Close stops the router and cleans up resources.
func (r *UserRouter) Close() error {
	r.logger.Info("Closing UserRouter")
	return r.Router.Close()
}
