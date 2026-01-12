package scorerouter

import (
	"context"
	"fmt"
	"log/slog"

	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	sharedscoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/score"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// ScoreRouter handles routing for score module events.
type ScoreRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewScoreRouter creates a new ScoreRouter.
func NewScoreRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *ScoreRouter {
	return &ScoreRouter{
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

// handlerDeps groups dependencies needed for handler registration
type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// registerHandler registers a pure transformation-pattern handler with typed payload
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-score." + topic

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
func (r *ScoreRouter) Configure(ctx context.Context, handlers scorehandlers.Handlers) error {
	r.logger.InfoContext(ctx, "ScoreRouter.Configure called")
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-score"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	err := r.RegisterHandlers(ctx, handlers)
	if err != nil {
		r.logger.ErrorContext(ctx, "ScoreRouter.RegisterHandlers failed", attr.Error(err))
		return fmt.Errorf("failed to register score handlers: %w", err)
	}
	r.logger.InfoContext(ctx, "ScoreRouter.Configure completed successfully")
	return nil
}

// RegisterHandlers registers event handlers.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	r.logger.InfoContext(ctx, "ScoreRouter.RegisterHandlers called")

	var metrics handlerwrapper.ReturningMetrics // reserved for Phase 6

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	// Register typed handlers
	registerHandler(deps, sharedscoreevents.ScoreUpdateRequestDiscordV1, handlers.HandleScoreUpdateRequestTyped)
	registerHandler(deps, scoreevents.ScoreUpdatedV1, handlers.HandleScoreUpdateSuccessTyped)
	registerHandler(deps, scoreevents.ScoreUpdateFailedV1, handlers.HandleScoreUpdateFailureTyped)
	// Optional: map backend score processing failures for observability (no downstream messages)
	registerHandler(deps, scoreevents.ProcessRoundScoresFailedV1, handlers.HandleProcessRoundScoresFailedTyped)

	r.logger.InfoContext(ctx, "ScoreRouter.RegisterHandlers completed successfully")
	return nil
}

// Close stops the router.
func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
