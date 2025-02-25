package userrouter

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger           observability.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	discord          discord.Operations
	config           *config.Config
	helper           utils.Helpers
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, discord discord.Operations, config *config.Config, helper utils.Helpers, tracer observability.Tracer) *UserRouter {
	return &UserRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		discord:          discord,
		config:           config,
		helper:           helper,
		tracer:           tracer,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router.
func (r *UserRouter) Configure(handlers userhandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		r.tracer.TraceHandler,
		observability.LokiLoggingMiddleware(r.logger),
	)

	if err := r.RegisterHandlers(context.Background(), handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	return nil
}

// RegisterHandlers registers event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		userevents.UserRoleUpdated:              handlers.HandleRoleUpdateResult,
		userevents.UserRoleUpdateFailed:         handlers.HandleRoleUpdateResult,
		userevents.UserCreated:                  handlers.HandleUserCreated,
		userevents.UserCreationFailed:           handlers.HandleUserCreationFailed,
		discorduserevents.RoleUpdateCommand:     handlers.HandleRoleUpdateCommand,
		discorduserevents.RoleUpdateButtonPress: handlers.HandleRoleUpdateButtonPress,
		userevents.UserSignupRequest:            handlers.HandleUserSignupRequest,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord.user.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			topic,
			r.publisher,
			handlerFunc,
		)
	}
	return nil
}

// Close stops the router.
func (r *UserRouter) Close() error {
	return r.Router.Close()
}
