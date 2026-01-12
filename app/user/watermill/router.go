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
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

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
	metrics          discordmetrics.DiscordMetrics
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
	metrics discordmetrics.DiscordMetrics,
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
		metrics:          metrics,
	}
}

// registerHandler registers a pure transformation-pattern handler with typed payload
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-user." + topic

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"", // Watermill reads topic from message metadata when empty
		deps.publisher,
		handlerwrapper.WrapTransformingTyped(
			handlerName,
			deps.logger,
			deps.tracer,
			deps.helper,
			deps.metrics,
			handler,
		),
	)
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

	if err := r.registerHandlers(handlers); err != nil {
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
		r.logger.Warn("unknown handler in topic resolution - no metadata fallback in Phase 2",
			attr.String("handler", handlerName),
		)
		return ""
	}
}

// registerHandlers registers all user module handlers using the generic pattern
func (r *UserRouter) registerHandlers(handlers userhandlers.Handler) error {
	var metrics handlerwrapper.ReturningMetrics // reserved for metrics integration

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	// Register all user module handlers
	registerHandler(deps, userevents.UserCreatedV1, handlers.HandleUserCreated)
	registerHandler(deps, userevents.UserCreationFailedV1, handlers.HandleUserCreationFailed)
	registerHandler(deps, shareduserevents.SignupAddRoleV1, handlers.HandleAddRole)
	registerHandler(deps, shareduserevents.SignupRoleAddedV1, handlers.HandleRoleAdded)
	registerHandler(deps, shareduserevents.SignupRoleAdditionFailedV1, handlers.HandleRoleAdditionFailed)
	registerHandler(deps, shareduserevents.RoleUpdateCommandV1, handlers.HandleRoleUpdateCommand)
	registerHandler(deps, shareduserevents.RoleUpdateButtonPressV1, handlers.HandleRoleUpdateButtonPress)
	registerHandler(deps, userevents.UserRoleUpdatedV1, handlers.HandleRoleUpdated)
	registerHandler(deps, userevents.UserRoleUpdateFailedV1, handlers.HandleRoleUpdateFailed)

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
