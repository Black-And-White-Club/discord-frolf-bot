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

// Configure sets up the router.
func (r *ScoreRouter) Configure(ctx context.Context, handlers scorehandlers.Handlers) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-score"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to configure score router: %w", err)
	}
	return nil
}

// handlerDeps groups dependencies needed for handler registration
type handlerDeps struct {
	router     *message.Router
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
}

// registerHandler registers a pure transformation-pattern handler with typed payload
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := fmt.Sprintf("discord-score.%s", topic)

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
}

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// This centralizes routing logic in the router (not in handlers or helpers).
func (r *ScoreRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Extract base topic from handlerName format: "discord-score.{topic}"
	// Map handler input topic → output topic(s)

	switch {
	case handlerName == "discord-score."+sharedscoreevents.ScoreUpdateRequestDiscordV1:
		// HandleScoreUpdateRequest always returns ScoreUpdateRequestedV1
		return scoreevents.ScoreUpdateRequestedV1

	case handlerName == "discord-score."+scoreevents.ScoreUpdatedV1:
		// HandleScoreUpdateSuccess always returns ScoreUpdateResponseDiscordV1
		return sharedscoreevents.ScoreUpdateResponseDiscordV1

	case handlerName == "discord-score."+scoreevents.ScoreUpdateFailedV1:
		// HandleScoreUpdateFailure returns either ScoreUpdateFailedDiscordV1 or nil (suppressed)
		// Phase 2: no metadata fallback; return empty to indicate unresolved routing.
		r.logger.Warn("score update failure handler has conditional outputs; no metadata fallback in Phase 2",
			attr.String("handler", handlerName),
		)
		return ""

	case handlerName == "discord-score."+scoreevents.ProcessRoundScoresFailedV1:
		// HandleProcessRoundScoresFailed doesn't return messages (nil)
		return ""

	default:
		r.logger.Warn("unknown handler in topic resolution",
			attr.String("handler", handlerName),
		)
		// Fallback to metadata (graceful degradation during migration)
		return msg.Metadata.Get("topic")
	}
}

// RegisterHandlers registers event handlers.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Score Handlers")

	// Use the returning wrapper so handlers can be pure transformers that return []handlerwrapper.Result
	deps := handlerDeps{
		router:     r.Router,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    nil,
		subscriber: r.subscriber,
		publisher:  r.publisher,
	}

	// Register typed handlers
	registerHandler(deps, sharedscoreevents.ScoreUpdateRequestDiscordV1, handlers.HandleScoreUpdateRequestTyped)
	registerHandler(deps, scoreevents.ScoreUpdatedV1, handlers.HandleScoreUpdateSuccessTyped)
	registerHandler(deps, scoreevents.ScoreUpdateFailedV1, handlers.HandleScoreUpdateFailureTyped)

	return nil
}

// Close stops the router.
func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
