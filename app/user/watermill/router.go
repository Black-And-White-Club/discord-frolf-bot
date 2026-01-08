package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	shareduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
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

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// This centralizes routing logic in the router (not in handlers or helpers).
func (r *UserRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Extract base topic from handlerName format: "discord-user.{topic}"
	// Map handler input topic → output topic(s)

	switch {
	case handlerName == "discord-user."+userevents.UserCreatedV1:
		// HandleUserCreated always returns SignupAddRoleV1
		return shareduserevents.SignupAddRoleV1

	case handlerName == "discord-user."+shareduserevents.SignupAddRoleV1:
		// HandleAddRole returns either SignupRoleAddedV1 or SignupRoleAdditionFailedV1
		// Check metadata for result (fallback to metadata temporarily for complex case)
		return msg.Metadata.Get("topic")

	case handlerName == "discord-user."+shareduserevents.SignupRoleAddedV1:
		// HandleRoleAdded doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+shareduserevents.SignupRoleAdditionFailedV1:
		// HandleRoleAdditionFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+userevents.UserCreationFailedV1:
		// HandleUserCreationFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+userevents.UserRoleUpdatedV1:
		// HandleRoleUpdated doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+userevents.UserRoleUpdateFailedV1:
		// HandleRoleUpdateFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+shareduserevents.RoleUpdateCommandV1:
		// HandleRoleUpdateCommand doesn't return messages (nil)
		return ""

	case handlerName == "discord-user."+shareduserevents.RoleUpdateButtonPressV1:
		// HandleRoleUpdateButtonPress always returns UserRoleUpdateRequestedV1
		return userevents.UserRoleUpdateRequestedV1

	default:
		r.logger.Warn("unknown handler in topic resolution",
			attr.String("handler", handlerName),
		)
		// Fallback to metadata (graceful degradation during migration)
		return msg.Metadata.Get("topic")
	}
}

// RegisterHandlers wires all event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handler) error {
	r.logger.InfoContext(ctx, "Registering User Handlers")

	eventsToHandlers := map[string]message.HandlerFunc{
		userevents.UserRoleUpdatedV1:               handlers.HandleRoleUpdated,
		userevents.UserRoleUpdateFailedV1:          handlers.HandleRoleUpdateFailed,
		userevents.UserCreatedV1:                   handlers.HandleUserCreated,
		shareduserevents.SignupAddRoleV1:           handlers.HandleAddRole,
		shareduserevents.SignupRoleAddedV1:         handlers.HandleRoleAdded,
		shareduserevents.SignupRoleAdditionFailedV1: handlers.HandleRoleAdditionFailed,
		userevents.UserCreationFailedV1:            handlers.HandleUserCreationFailed,
		shareduserevents.RoleUpdateCommandV1:       handlers.HandleRoleUpdateCommand,
		shareduserevents.RoleUpdateButtonPressV1:   handlers.HandleRoleUpdateButtonPress,
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
					// Router resolves topic (not metadata)
					publishTopic := r.getPublishTopic(handlerName, m)

					// INVARIANT: Topic must be resolvable
					if publishTopic == "" {
						r.logger.Error("router failed to resolve publish topic - MESSAGE DROPPED",
							attr.String("handler", handlerName),
							attr.String("msg_uuid", m.UUID),
							attr.String("correlation_id", m.Metadata.Get("correlation_id")),
						)
						// Skip publishing but don't fail entire batch
						continue
					}

					r.logger.InfoContext(ctx, "Publishing message",
						attr.String("topic", publishTopic),
						attr.String("handler", handlerName),
						attr.String("correlation_id", m.Metadata.Get("correlation_id")),
					)

					if err := r.publisher.Publish(publishTopic, m); err != nil {
						return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
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
