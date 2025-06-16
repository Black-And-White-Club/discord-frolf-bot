package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewRoundRouter creates a new RoundRouter.
func NewRoundRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *RoundRouter {
	return &RoundRouter{
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
func (r *RoundRouter) Configure(ctx context.Context, handlers roundhandlers.Handlers) error {
	// Add Prometheus metrics
	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheus.NewRegistry(), "", "")
	metricsBuilder.AddPrometheusRouterMetrics(r.Router)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-round"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register round handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Round Handlers")

	eventsToHandlers := map[string]message.HandlerFunc{
		// Creation flow
		discordroundevents.RoundCreateModalSubmit: handlers.HandleRoundCreateRequested,
		roundevents.RoundCreated:                  handlers.HandleRoundCreated,
		roundevents.RoundCreationFailed:           handlers.HandleRoundCreationFailed,
		roundevents.RoundValidationFailed:         handlers.HandleRoundValidationFailed,

		// Update flow
		discordroundevents.RoundUpdateRequestTopic: handlers.HandleRoundUpdateRequested,
		roundevents.RoundScheduleUpdate:            handlers.HandleRoundUpdated,
		roundevents.RoundScheduled:                 handlers.HandleRoundUpdateFailed,

		// Participation
		discordroundevents.RoundParticipantJoinReqTopic: handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantRemoved:             handlers.HandleRoundParticipantRemoved,

		// Scoring
		roundevents.DiscordParticipantScoreUpdated: handlers.HandleParticipantScoreUpdated,
		roundevents.RoundScoreUpdateError:          handlers.HandleScoreUpdateError,

		// Lifecycle
		discordroundevents.RoundDeletedTopic: handlers.HandleRoundDeleted,
		roundevents.DiscordRoundFinalized:    handlers.HandleRoundFinalized,
		roundevents.DiscordRoundStarted:      handlers.HandleRoundStarted,

		// Tag handling
		roundevents.RoundParticipantJoined: handlers.HandleRoundParticipantJoined,

		// Reminders
		roundevents.DiscordRoundReminder: handlers.HandleRoundReminder,

		roundevents.TagsUpdatedForScheduledRounds: handlers.HandleTagsUpdatedForScheduledRounds,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-round.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message",
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
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
