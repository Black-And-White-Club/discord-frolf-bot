package leaderboardrouter

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/events/leaderboard"
	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// LeaderboardRouter handles routing for leaderboard module events.
type LeaderboardRouter struct {
	logger           observability.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	session          discord.Session
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, session discord.Session, tracer observability.Tracer) *LeaderboardRouter {
	return &LeaderboardRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		session:          session,
		tracer:           tracer,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router.
func (r *LeaderboardRouter) Configure(handlers leaderboardhandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("leaderboard"),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		r.tracer.TraceHandler,
		observability.LokiLoggingMiddleware(r.logger),
	)

	if err := r.RegisterHandlers(context.Background(), handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	return nil
}

// RegisterHandlers registers event handlers.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		discordleaderboardevents.LeaderboardRetrieveRequestTopic:         handlers.HandleLeaderboardRetrieveRequest,
		discordleaderboardevents.LeaderboardTagAssignRequestTopic:        handlers.HandleTagAssignRequest,
		discordleaderboardevents.LeaderboardTagAvailabilityRequestTopic:  handlers.HandleGetTagByDiscordID,
		discordleaderboardevents.LeaderboardTagSwapRequestTopic:          handlers.HandleTagSwapRequest,
		leaderboardevents.LeaderboardUpdated:                             handlers.HandleLeaderboardData,
		leaderboardevents.GetLeaderboardResponse:                         handlers.HandleLeaderboardData,
		discordleaderboardevents.LeaderboardTagAssignedTopic:             handlers.HandleTagAssignedResponse,
		discordleaderboardevents.LeaderboardTagAssignFailedTopic:         handlers.HandleTagAssignFailedResponse,
		discordleaderboardevents.LeaderboardTagAvailabilityResponseTopic: handlers.HandleGetTagByDiscordIDResponse,
		discordleaderboardevents.LeaderboardTagSwappedTopic:              handlers.HandleTagSwappedResponse,
		discordleaderboardevents.LeaderboardTagSwapFailedTopic:           handlers.HandleTagSwapFailedResponse,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord.leaderboard.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			topic,
			r.publisher,
			handlerFunc,
		)
	}
	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
