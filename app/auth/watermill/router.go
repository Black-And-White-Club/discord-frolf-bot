package authwatermill

import (
	"context"
	"log/slog"

	authhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/auth/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
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

// AuthRouter handles auth-related Watermill routing.
type AuthRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewAuthRouter creates a new AuthRouter.
func NewAuthRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *AuthRouter {
	return &AuthRouter{
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

// Configure sets up the auth router with handlers.
func (r *AuthRouter) Configure(ctx context.Context, handlers *authhandlers.AuthHandlers) error {
	r.logger.InfoContext(ctx, "Configuring Auth Watermill router")

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-auth"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.registerHandlers(handlers); err != nil {
		return err
	}

	r.logger.InfoContext(ctx, "Auth Watermill router configured")
	return nil
}

func (r *AuthRouter) registerHandlers(handlers *authhandlers.AuthHandlers) error {
	var metrics handlerwrapper.ReturningMetrics

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	registerHandler(deps, authevents.MagicLinkGeneratedV1, handlers.HandleMagicLinkGenerated)

	return nil
}

func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-auth." + topic

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"",
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

	deps.logger.Info("Registered Auth handler",
		attr.String("handler", handlerName),
		attr.String("topic", topic),
	)
}

// Close closes the auth router.
func (r *AuthRouter) Close() error {
	return r.Router.Close()
}
