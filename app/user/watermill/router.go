package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/user"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
	userDiscord      interface{} // Store userDiscord for access to signup manager
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *UserRouter {
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
func (r *UserRouter) Configure(ctx context.Context, handlers userhandlers.Handler) error {
	// Note: Using Discord metrics instead of separate Prometheus metrics to avoid conflicts

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register user handlers: %w", err)
	}
	return nil
}

// RegisterHandlers wires all event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handler) error {
	r.logger.InfoContext(ctx, "Registering User Handlers")

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

		// Use environment-specific queue groups for multi-tenant scalability
		// This ensures only one instance processes each message per environment
		queueGroup := fmt.Sprintf("user-handlers-%s", r.config.Observability.Environment)

		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			queueGroup,
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing user message",
						attr.String("message_id", msg.UUID),
						attr.Error(err),
					)
					return nil, err
				}
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						r.logger.InfoContext(ctx, "Publishing message",
							attr.String("message_id", m.UUID),
							attr.String("topic", publishTopic),
						)
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						r.logger.WarnContext(ctx, "Message missing topic metadata",
							attr.String("message_id", m.UUID),
						)
					}
				}
				return nil, nil
			},
		)
	}
	return nil
}

// Close gracefully stops the router.
func (r *UserRouter) Close() error {
	return r.Router.Close()
}

// SetUserDiscord stores the user discord module for access to signup manager.
func (r *UserRouter) SetUserDiscord(ud userdiscord.UserDiscordInterface) {
	r.userDiscord = ud
}

// GetSignupManager returns the signup manager from the user discord module.
func (r *UserRouter) GetSignupManager() signup.SignupManager {
	if ud, ok := r.userDiscord.(userdiscord.UserDiscordInterface); ok {
		return ud.GetSignupManager()
	}
	return nil
}
