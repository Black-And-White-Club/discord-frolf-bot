package scorerouter

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/score/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
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
func (r *ScoreRouter) Configure(
	session discord.Session,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	discordMetrics discordmetrics.DiscordMetrics,
	eventbus eventbus.EventBus,
) error {
	// Create Prometheus metrics builder
	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheus.NewRegistry(), "", "")
	// Add metrics middleware to the router
	metricsBuilder.AddPrometheusRouterMetrics(r.Router)

	// Create Score Handlers with their dependencies
	scoreHandlers := scorehandlers.NewScoreHandlers(r.logger, config, session, helper, tracer, discordMetrics)

	// Add middleware specific to the score module
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-score"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(context.Background(), scoreHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handler) error {
	r.logger.InfoContext(ctx, "Entering Register Handlers for Score")

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
			"",  // No direct publish topic
			nil, // No manual publisher
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						r.logger.InfoContext(ctx, "🚀 Auto-publishing message", attr.String("message_id", m.UUID), attr.String("topic", publishTopic))
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						r.logger.Warn("⚠️ Message missing topic metadata, dropping", attr.String("message_id", m.UUID))
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
