package userrouter

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	tempo "github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger     observability.Logger
	Router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	session    discord.Session
	tracer     tempo.Tracer
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, session discord.Session, tracer tempo.Tracer) *UserRouter {
	return &UserRouter{
		logger:     logger,
		Router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		session:    session,
		tracer:     tracer,
	}
}

// Configure sets up the router.
func (r *UserRouter) Configure(handlers userhandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		middleware.Retry{
			MaxRetries: 3,
		}.Middleware,
		r.tracer.TraceHandler,
		r.LokiLoggingMiddleware,
	)

	if err := r.RegisterHandlers(context.Background(), handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	return nil
}

// LokiLoggingMiddleware is the custom Watermill middleware.
func (r *UserRouter) LokiLoggingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		startTime := time.Now()
		ctx := msg.Context()

		handlerName := msg.Metadata.Get("handler_name")
		domain := msg.Metadata.Get("domain")

		r.logger.Info(ctx, "Received message",
			attr.CorrelationIDFromMsg(msg),
			attr.Topic(msg.Metadata.Get("topic")),
			attr.MessageID(msg),
			attr.String("handler", handlerName),
			attr.String("domain", domain),
		)

		for key, value := range msg.Metadata {
			r.logger.Info(ctx, "Message metadata", attr.String(key, value))
		}

		producedMessages, err := next(msg)

		duration := time.Since(startTime)

		if err != nil {
			r.logger.Error(ctx, "Error processing message",
				attr.CorrelationIDFromMsg(msg),
				attr.Topic(msg.Metadata.Get("topic")),
				attr.MessageID(msg),
				attr.Duration("duration", duration),
				attr.String("handler", handlerName),
				attr.String("domain", domain),
				attr.Error(err),
			)
		} else {
			r.logger.Info(ctx, "Message processed successfully",
				attr.CorrelationIDFromMsg(msg),
				attr.Topic(msg.Metadata.Get("topic")),
				attr.MessageID(msg),
				attr.Duration("duration", duration),
				attr.String("handler", handlerName),
				attr.String("domain", domain),
			)
		}

		return producedMessages, err

	}
}

const ()

// RegisterHandlers registers event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		userevents.UserRoleUpdated:      handlers.HandleRoleUpdateResult,
		userevents.UserRoleUpdateFailed: handlers.HandleRoleUpdateResult,
		userevents.UserCreated:          handlers.HandleUserCreated,
		userevents.UserCreationFailed:   handlers.HandleUserCreationFailed,
		discorduserevents.SendUserDM:    handlers.HandleSendUserDM,
		discorduserevents.DMSent:        handlers.HandleDMSent,
		discorduserevents.DMCreateError: handlers.HandleDMCreateError,
		discorduserevents.DMSendError:   handlers.HandleDMSendError,
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
