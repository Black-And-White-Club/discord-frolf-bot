package router

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
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
	r.logger.InfoContext(ctx, "LeaderboardRouter.Configure called")
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register leaderboard handlers: %w", err)
	}
	r.logger.InfoContext(ctx, "LeaderboardRouter.Configure completed successfully")
	return nil
}

// handlerDeps groups dependencies needed for handler registration.
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
	handlerName := "discord-leaderboard." + topic

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

// RegisterHandlers registers event handlers with type-safe payload handling.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "LeaderboardRouter.RegisterHandlers called")

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

	// Tag Assignment
	registerHandler(deps, discordleaderboardevents.LeaderboardTagAssignRequestV1, handlers.HandleTagAssignRequest)
	registerHandler(deps, leaderboardevents.LeaderboardTagAssignedV1, handlers.HandleTagAssignedResponse)
	registerHandler(deps, leaderboardevents.LeaderboardTagAssignmentFailedV1, handlers.HandleTagAssignFailedResponse)

	// Tag Swap
	registerHandler(deps, discordleaderboardevents.LeaderboardTagSwapRequestV1, handlers.HandleTagSwapRequest)
	registerHandler(deps, leaderboardevents.TagSwapProcessedV1, handlers.HandleTagSwappedResponse)
	registerHandler(deps, leaderboardevents.TagSwapFailedV1, handlers.HandleTagSwapFailedResponse)

	// Tag Lookup
	registerHandler(deps, discordleaderboardevents.LeaderboardTagAvailabilityRequestV1, handlers.HandleGetTagByDiscordID)
	registerHandler(deps, sharedevents.GetTagNumberResponseV1, handlers.HandleGetTagByDiscordIDResponse)
	registerHandler(deps, sharedevents.GetTagNumberFailedV1, handlers.HandleGetTagByDiscordIDFailed)

	// Leaderboard Retrieval
	registerHandler(deps, discordleaderboardevents.LeaderboardRetrieveRequestV1, handlers.HandleLeaderboardRetrieveRequest)
	registerHandler(deps, leaderboardevents.LeaderboardUpdatedV1, handlers.HandleLeaderboardUpdatedNotification)
	registerHandler(deps, leaderboardevents.GetLeaderboardResponseV1, handlers.HandleLeaderboardResponse)

	// Leaderboard Updates
	registerHandler(deps, leaderboardevents.LeaderboardBatchTagAssignedV1, handlers.HandleBatchTagAssigned)

	// Leaderboard Errors
	registerHandler(deps, leaderboardevents.LeaderboardUpdateFailedV1, handlers.HandleLeaderboardUpdateFailed)
	registerHandler(deps, leaderboardevents.GetLeaderboardFailedV1, handlers.HandleLeaderboardRetrievalFailed)

	r.logger.InfoContext(ctx, "LeaderboardRouter.RegisterHandlers completed successfully")
	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
