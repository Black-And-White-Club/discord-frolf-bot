package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
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

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// This centralizes routing logic in the router (not in handlers or helpers).
func (r *LeaderboardRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Extract base topic from handlerName format: "discord-leaderboard.{topic}"
	// Map handler input topic → output topic(s)

	switch {
	case handlerName == "discord-leaderboard."+sharedleaderboardevents.LeaderboardTagAssignRequestV1:
		// HandleTagAssignRequest always returns LeaderboardBatchTagAssignmentRequestedV1
		return leaderboardevents.LeaderboardBatchTagAssignmentRequestedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.LeaderboardTagAssignedV1:
		// HandleTagAssignedResponse always returns LeaderboardTagAssignedV1
		return sharedleaderboardevents.LeaderboardTagAssignedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.LeaderboardTagAssignmentFailedV1:
		// HandleTagAssignFailedResponse always returns LeaderboardTagAssignFailedV1
		return sharedleaderboardevents.LeaderboardTagAssignFailedV1

	case handlerName == "discord-leaderboard."+sharedleaderboardevents.LeaderboardTagAvailabilityRequestV1:
		// HandleGetTagByDiscordID always returns GetTagByUserIDRequestedV1
		return leaderboardevents.GetTagByUserIDRequestedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.GetTagNumberResponseV1:
		// HandleGetTagByDiscordIDResponse returns LeaderboardTagAvailabilityResponseV1 or nil (conditional)
		// Check metadata for result (fallback to metadata temporarily for conditional case)
		return msg.Metadata.Get("topic")

	case handlerName == "discord-leaderboard."+leaderboardevents.LeaderboardBatchTagAssignedV1:
		// HandleBatchTagAssigned always returns LeaderboardTraceEvent
		return leaderboardevents.LeaderboardTraceEvent

	case handlerName == "discord-leaderboard."+sharedleaderboardevents.LeaderboardRetrieveRequestV1:
		// HandleLeaderboardRetrieveRequest always returns GetLeaderboardRequestedV1
		return leaderboardevents.GetLeaderboardRequestedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.GetLeaderboardResponseV1:
		// HandleLeaderboardData returns LeaderboardRetrievedV1
		return sharedleaderboardevents.LeaderboardRetrievedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.LeaderboardUpdatedV1:
		// HandleLeaderboardData (when topic is LeaderboardUpdated) returns GetLeaderboardRequestedV1
		return leaderboardevents.GetLeaderboardRequestedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.LeaderboardUpdateFailedV1:
		// HandleLeaderboardUpdateFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-leaderboard."+leaderboardevents.GetLeaderboardFailedV1:
		// HandleLeaderboardRetrievalFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-leaderboard."+sharedleaderboardevents.LeaderboardTagSwapRequestV1:
		// HandleTagSwapRequest always returns TagSwapRequestedV1
		return leaderboardevents.TagSwapRequestedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.TagSwapProcessedV1:
		// HandleTagSwappedResponse always returns LeaderboardTagSwappedV1
		return sharedleaderboardevents.LeaderboardTagSwappedV1

	case handlerName == "discord-leaderboard."+leaderboardevents.TagSwapFailedV1:
		// HandleTagSwapFailedResponse always returns LeaderboardTagSwapFailedV1
		return sharedleaderboardevents.LeaderboardTagSwapFailedV1

	default:
		r.logger.Warn("unknown handler in topic resolution",
			attr.String("handler", handlerName),
		)
		// Fallback to metadata (graceful degradation during migration)
		return msg.Metadata.Get("topic")
	}
}

// RegisterHandlers registers event handlers.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		// Tag management
		sharedleaderboardevents.LeaderboardTagAssignRequestV1: handlers.HandleTagAssignRequest,
		leaderboardevents.LeaderboardTagAssignedV1:            handlers.HandleTagAssignedResponse,
		leaderboardevents.LeaderboardTagAssignmentFailedV1:    handlers.HandleTagAssignFailedResponse,

		// Tag lookup
		sharedleaderboardevents.LeaderboardTagAvailabilityRequestV1: handlers.HandleGetTagByDiscordID,
		leaderboardevents.GetTagNumberResponseV1:                    handlers.HandleGetTagByDiscordIDResponse,

		// Leaderboard updates
		leaderboardevents.LeaderboardBatchTagAssignedV1:      handlers.HandleBatchTagAssigned,
		sharedleaderboardevents.LeaderboardRetrieveRequestV1: handlers.HandleLeaderboardRetrieveRequest,
		leaderboardevents.GetLeaderboardResponseV1:           handlers.HandleLeaderboardData,
		leaderboardevents.LeaderboardUpdatedV1:               handlers.HandleLeaderboardData,

		// Leaderboard errors
		leaderboardevents.LeaderboardUpdateFailedV1: handlers.HandleLeaderboardUpdateFailed,
		leaderboardevents.GetLeaderboardFailedV1:    handlers.HandleLeaderboardRetrievalFailed,

		// Tag swaps
		sharedleaderboardevents.LeaderboardTagSwapRequestV1: handlers.HandleTagSwapRequest,
		leaderboardevents.TagSwapProcessedV1:                handlers.HandleTagSwappedResponse,
		leaderboardevents.TagSwapFailedV1:                   handlers.HandleTagSwapFailedResponse,
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
					// Router resolves topic (not metadata)
					publishTopic := r.getPublishTopic(handlerName, m)

					// INVARIANT: Topic must be resolvable
					if publishTopic == "" {
						r.logger.Error("router failed to resolve publish topic - MESSAGE DROPPED",
							attr.String("handler", handlerName),
							attr.String("msg_uuid", m.UUID),
							attr.String("correlation_id", m.Metadata.Get("correlation_id")),
						)
						// Skip publishing but don't fail entire batch
						continue
					}

					r.logger.InfoContext(ctx, "Publishing message",
						attr.String("topic", publishTopic),
						attr.String("handler", handlerName),
						attr.String("correlation_id", m.Metadata.Get("correlation_id")),
					)

					if err := r.publisher.Publish(publishTopic, m); err != nil {
						return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
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
