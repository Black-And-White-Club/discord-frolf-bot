package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/leaderboard"
	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardRouter handles routing for leaderboard module events.
type LeaderboardRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *LeaderboardRouter {
	return &LeaderboardRouter{
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
func (r *LeaderboardRouter) Configure(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	// Note: Using Discord metrics instead of separate Prometheus metrics to avoid conflicts

	// Add middleware
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		// Tag management
		discordleaderboardevents.LeaderboardTagAssignRequestTopic: handlers.HandleTagAssignRequest,
		leaderboardevents.LeaderboardTagAssignmentSuccess:         handlers.HandleTagAssignedResponse,
		leaderboardevents.LeaderboardTagAssignmentFailed:          handlers.HandleTagAssignFailedResponse,

		// Tag lookup
		discordleaderboardevents.LeaderboardTagAvailabilityRequestTopic: handlers.HandleGetTagByDiscordID,
		leaderboardevents.GetTagNumberResponse:                          handlers.HandleGetTagByDiscordIDResponse,

		// Leaderboard updates
		leaderboardevents.LeaderboardBatchTagAssigned:      handlers.HandleBatchTagAssigned,
		discordleaderboardevents.LeaderboardRetrievedTopic: handlers.HandleLeaderboardData,

		// Tag swaps
		discordleaderboardevents.LeaderboardTagSwapRequestTopic: handlers.HandleTagSwapRequest,
		leaderboardevents.TagSwapProcessed:                      handlers.HandleTagSwappedResponse,
		leaderboardevents.TagSwapFailed:                         handlers.HandleTagSwapFailedResponse,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-leaderboard.%s", topic)

		// Use environment-specific queue groups for multi-tenant scalability
		// This ensures only one instance processes each message per environment
		queueGroup := fmt.Sprintf("leaderboard-handlers-%s", r.config.Observability.Environment)

		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			queueGroup,
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
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
