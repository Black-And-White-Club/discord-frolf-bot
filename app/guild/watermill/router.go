package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// GuildRouter handles routing for guild module events.
type GuildRouter struct {
	logger     *slog.Logger
	Router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	config     *config.Config
	helper     utils.Helpers
	tracer     trace.Tracer
}

// NewGuildRouter creates a new GuildRouter.
func NewGuildRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *GuildRouter {
	return &GuildRouter{
		logger:     logger,
		Router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		config:     config,
		helper:     helper,
		tracer:     tracer,
	}
}

// Configure sets up the router.
func (r *GuildRouter) Configure(ctx context.Context, handlers guildhandlers.Handlers) error {
	// Note: Using Discord metrics instead of separate Prometheus metrics to avoid conflicts

	middlewareHelper := utils.NewMiddlewareHelper()
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		middlewareHelper.CommonMetadataMiddleware("discord-guild"),
		middlewareHelper.DiscordMetadataMiddleware(),
		middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register guild handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers using pure transformation pattern.
func (r *GuildRouter) RegisterHandlers(ctx context.Context, handlers guildhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Guild Handlers")

	var metrics handlerwrapper.ReturningMetrics // reserved for future metrics integration

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	// Guild config creation/setup flow - affects command registration
	registerHandler(deps, guildevents.GuildConfigCreatedV1, handlers.HandleGuildConfigCreated)
	registerHandler(deps, guildevents.GuildConfigCreationFailedV1, handlers.HandleGuildConfigCreationFailed)

	// Guild config update flow - may affect command permissions
	registerHandler(deps, guildevents.GuildConfigUpdatedV1, handlers.HandleGuildConfigUpdated)
	registerHandler(deps, guildevents.GuildConfigUpdateFailedV1, handlers.HandleGuildConfigUpdateFailed)

	// Guild config retrieval flow - informational only, no command action
	registerHandler(deps, guildevents.GuildConfigRetrievedV1, handlers.HandleGuildConfigRetrieved)
	registerHandler(deps, guildevents.GuildConfigRetrievalFailedV1, handlers.HandleGuildConfigRetrievalFailed)

	// Guild config deletion flow - affects command registration
	registerHandler(deps, guildevents.GuildConfigDeletedV1, handlers.HandleGuildConfigDeleted)
	registerHandler(deps, guildevents.GuildConfigDeletionFailedV1, handlers.HandleGuildConfigDeletionFailed)

	r.logger.InfoContext(ctx, "Guild router configured successfully")

	return nil
}

type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// registerHandler registers a pure transformation-pattern handler with typed payload.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-guild." + topic

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

// Close gracefully shuts down the router
func (r *GuildRouter) Close() error {
	if r.Router != nil {
		return r.Router.Close()
	}
	return nil
}
