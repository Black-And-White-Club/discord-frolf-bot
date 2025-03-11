package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
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
	config           *config.Config
	helper           utils.Helpers
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewUserRouter creates a new UserRouter.

func NewUserRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, config *config.Config, helper utils.Helpers, tracer observability.Tracer) *UserRouter {
	return &UserRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		config:           config,
		helper:           helper,
		tracer:           tracer,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router.
func (r *UserRouter) Configure(handlers userhandlers.UserHandler, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-user"),
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
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.UserHandler) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		discorduserevents.RoleResponse:             handlers.HandleRoleUpdateResult,
		discorduserevents.RoleResponseFailed:       handlers.HandleRoleUpdateResult,
		discorduserevents.SignupSuccess:            handlers.HandleUserCreated,
		discorduserevents.SignupAddRole:            handlers.HandleAddRole,
		discorduserevents.SignupRoleAdded:          handlers.HandleRoleAdded,
		discorduserevents.SignupRoleAdditionFailed: handlers.HandleRoleAdditionFailed,
		discorduserevents.SignupFailed:             handlers.HandleUserCreationFailed,
		discorduserevents.RoleUpdateCommand:        handlers.HandleRoleUpdateCommand,
		discorduserevents.RoleUpdateButtonPress:    handlers.HandleRoleUpdateButtonPress,
		discorduserevents.SignupFormSubmitted:      handlers.HandleUserSignupRequest,
	}
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-user.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",  // ❌ No direct publish topic
			nil, // ❌ No manual publisher
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					return nil, err
				}
				// Automatically publish messages based on metadata
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						slog.Info("🚀 Auto-publishing message",
							slog.String("message_id", m.UUID),
							slog.String("topic", publishTopic),
						)
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						slog.Warn("⚠️ Message missing topic metadata, dropping.",
							slog.String("message_id", m.UUID),
						)
					}
				}
				return nil, nil // ✅ No messages returned, they're published instead
			},
		)
	}
	return nil
}

// Close stops the router.
func (r *UserRouter) Close() error {
	return r.Router.Close()
}
