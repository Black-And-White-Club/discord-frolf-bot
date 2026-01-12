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
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
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
	metrics          discordmetrics.DiscordMetrics
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
	metrics discordmetrics.DiscordMetrics,
) *LeaderboardRouter {
	return &LeaderboardRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		config:           config,
		helper:           helper,
		tracer:           tracer,
		metrics:          metrics,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router.
func (r *LeaderboardRouter) Configure(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
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

// createTypedHandler is a helper function to create a typed handler wrapper.
// It cannot be a method because Go doesn't support type parameters on methods.
func createTypedHandler[T any](
	logger *slog.Logger,
	tracer trace.Tracer,
	helper utils.Helpers,
	handlerName string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) message.HandlerFunc {
	return handlerwrapper.WrapTransformingTyped[T](
		handlerName,
		logger,
		tracer,
		helper,
		nil,
		handler,
	)
}

// RegisterHandlers registers event handlers with type-safe payload handling.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	// Map of topic → (payload type, handler function)
	type handlerRegistration struct {
		topic   string
		handler message.HandlerFunc
	}

	handlerRegistrations := []handlerRegistration{
		// Tag Assignment
		{
			topic: sharedleaderboardevents.LeaderboardTagAssignRequestV1,
			handler: createTypedHandler[sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagAssignRequest",
				func(ctx context.Context, payload *sharedleaderboardevents.LeaderboardTagAssignRequestPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagAssignRequest(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.LeaderboardTagAssignedV1,
			handler: createTypedHandler[leaderboardevents.LeaderboardTagAssignedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagAssignedResponse",
				func(ctx context.Context, payload *leaderboardevents.LeaderboardTagAssignedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagAssignedResponse(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.LeaderboardTagAssignmentFailedV1,
			handler: createTypedHandler[leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagAssignFailedResponse",
				func(ctx context.Context, payload *leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagAssignFailedResponse(ctx, payload)
				},
			),
		},

		// Tag Swap
		{
			topic: sharedleaderboardevents.LeaderboardTagSwapRequestV1,
			handler: createTypedHandler[sharedleaderboardevents.LeaderboardTagSwapRequestPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagSwapRequest",
				func(ctx context.Context, payload *sharedleaderboardevents.LeaderboardTagSwapRequestPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagSwapRequest(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.TagSwapProcessedV1,
			handler: createTypedHandler[leaderboardevents.TagSwapProcessedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagSwappedResponse",
				func(ctx context.Context, payload *leaderboardevents.TagSwapProcessedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagSwappedResponse(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.TagSwapFailedV1,
			handler: createTypedHandler[leaderboardevents.TagSwapFailedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleTagSwapFailedResponse",
				func(ctx context.Context, payload *leaderboardevents.TagSwapFailedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleTagSwapFailedResponse(ctx, payload)
				},
			),
		},

		// Tag Lookup
		{
			topic: sharedleaderboardevents.LeaderboardTagAvailabilityRequestV1,
			handler: createTypedHandler[sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleGetTagByDiscordID",
				func(ctx context.Context, payload *sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleGetTagByDiscordID(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.GetTagNumberResponseV1,
			handler: createTypedHandler[leaderboardevents.GetTagNumberResponsePayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleGetTagByDiscordIDResponse",
				func(ctx context.Context, payload *leaderboardevents.GetTagNumberResponsePayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleGetTagByDiscordIDResponse(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.GetTagNumberFailedV1,
			handler: createTypedHandler[leaderboardevents.GetTagNumberFailedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleGetTagByDiscordIDFailed",
				func(ctx context.Context, payload *leaderboardevents.GetTagNumberFailedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleGetTagByDiscordIDFailed(ctx, payload)
				},
			),
		},

		// Leaderboard Retrieval
		{
			topic: sharedleaderboardevents.LeaderboardRetrieveRequestV1,
			handler: createTypedHandler[sharedleaderboardevents.LeaderboardRetrieveRequestPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleLeaderboardRetrieveRequest",
				func(ctx context.Context, payload *sharedleaderboardevents.LeaderboardRetrieveRequestPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleLeaderboardRetrieveRequest(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.LeaderboardUpdatedV1,
			handler: createTypedHandler[leaderboardevents.LeaderboardUpdatedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleLeaderboardUpdatedNotification",
				func(ctx context.Context, payload *leaderboardevents.LeaderboardUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleLeaderboardUpdatedNotification(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.GetLeaderboardResponseV1,
			handler: createTypedHandler[leaderboardevents.GetLeaderboardResponsePayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleLeaderboardResponse",
				func(ctx context.Context, payload *leaderboardevents.GetLeaderboardResponsePayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleLeaderboardResponse(ctx, payload)
				},
			),
		},

		// Leaderboard Updates
		{
			topic: leaderboardevents.LeaderboardBatchTagAssignedV1,
			handler: createTypedHandler[leaderboardevents.LeaderboardBatchTagAssignedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleBatchTagAssigned",
				func(ctx context.Context, payload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleBatchTagAssigned(ctx, payload)
				},
			),
		},

		// Leaderboard Errors
		{
			topic: leaderboardevents.LeaderboardUpdateFailedV1,
			handler: createTypedHandler[leaderboardevents.LeaderboardUpdateFailedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleLeaderboardUpdateFailed",
				func(ctx context.Context, payload *leaderboardevents.LeaderboardUpdateFailedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleLeaderboardUpdateFailed(ctx, payload)
				},
			),
		},
		{
			topic: leaderboardevents.GetLeaderboardFailedV1,
			handler: createTypedHandler[leaderboardevents.GetLeaderboardFailedPayloadV1](
				r.logger,
				r.tracer,
				r.helper,
				"discord-leaderboard.HandleLeaderboardRetrievalFailed",
				func(ctx context.Context, payload *leaderboardevents.GetLeaderboardFailedPayloadV1) ([]handlerwrapper.Result, error) {
					return handlers.HandleLeaderboardRetrievalFailed(ctx, payload)
				},
			),
		},
	}

	for _, reg := range handlerRegistrations {
		handlerName := fmt.Sprintf("discord-leaderboard.%s", reg.topic)

		// Use environment-specific queue groups for multi-tenant scalability
		queueGroup := fmt.Sprintf("leaderboard-handlers-%s", r.config.Observability.Environment)

		r.Router.AddHandler(
			handlerName,
			reg.topic,
			r.subscriber,
			queueGroup,
			nil,
			reg.handler,
		)
	}

	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
