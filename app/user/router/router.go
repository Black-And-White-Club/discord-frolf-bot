package router

import (
	"context"
	"fmt"
	"log/slog"

	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/user/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discorduserevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/user"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
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
	userDiscord      any // Store userDiscord for access to signup manager
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
func (r *UserRouter) Configure(ctx context.Context, handlers userhandlers.Handlers) error {
	// Note: Using Discord metrics instead of separate Prometheus metrics to avoid conflicts

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.registerHandlers(handlers); err != nil {
		return fmt.Errorf("failed to register user handlers: %w", err)
	}
	return nil
}

// registerHandlers registers all user module handlers using the generic pattern
func (r *UserRouter) registerHandlers(handlers userhandlers.Handlers) error {
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
	registerHandler(deps, discorduserevents.SignupAddRoleV1, handlers.HandleAddRole)
	registerHandler(deps, discorduserevents.SignupRoleAddedV1, handlers.HandleRoleAdded)
	registerHandler(deps, discorduserevents.SignupRoleAdditionFailedV1, handlers.HandleRoleAdditionFailed)
	registerHandler(deps, discorduserevents.RoleUpdateCommandV1, handlers.HandleRoleUpdateCommand)
	registerHandler(deps, discorduserevents.RoleUpdateButtonPressV1, handlers.HandleRoleUpdateButtonPress)
	registerHandler(deps, userevents.UserRoleUpdatedV1, handlers.HandleRoleUpdated)
	registerHandler(deps, userevents.UserRoleUpdateFailedV1, handlers.HandleRoleUpdateFailed)
	registerHandler(deps, userevents.UserProfileSyncRequestTopicV1, handlers.HandleProfileSyncRequest)

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
