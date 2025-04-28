package scorerouter

import (
	"context"
	"fmt"
	"log/slog"

	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
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
func (r *ScoreRouter) Configure(ctx context.Context, handlers scorehandlers.Handler) error {
	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheus.NewRegistry(), "", "")
	metricsBuilder.AddPrometheusRouterMetrics(r.Router)

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

// RegisterHandlers registers event handlers.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handler) error {
	r.logger.InfoContext(ctx, "Registering Score Handlers")

	eventsToHandlers := map[string]message.HandlerFunc{
		scoreevents.ScoreUpdateRequest: handlers.HandleScoreUpdateRequest,
		scoreevents.ScoreUpdateSuccess: handlers.HandleScoreUpdateSuccess,
		scoreevents.ScoreUpdateFailure: handlers.HandleScoreUpdateFailure,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-score.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing score message",
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

// Close stops the router.
func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
