package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
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
	config           *config.Config
	helper           utils.Helpers
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, config *config.Config, helper utils.Helpers, tracer observability.Tracer) *LeaderboardRouter {
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
func (r *LeaderboardRouter) Configure(handlers leaderboardhandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Retry{MaxRetries: 3}.Middleware,
		r.middlewareHelper.CommonMetadataMiddleware("discord-leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
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
		leaderboardevents.LeaderboardUpdated: handlers.HandleLeaderboardUpdated,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-leaderboard.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",  // ❌ No direct publish topic
			nil, // ❌ No manual publisher
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					// Log the error and return it to trigger the retry logic
					slog.Error("Error processing message", slog.String("message_id", msg.UUID), attr.Error(err))
					return nil, err
				}
				// Automatically publish messages based on metadata
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						slog.Info("🚀 Auto-publishing message",
							slog.String("message_id", m.UUID),
							slog.String("topic", publishTopic),
						)
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						slog.Warn("⚠️ Message missing topic metadata, dropping.",
							slog.String("message_id", m.UUID),
						)
					}
				}
				return nil, nil // ✅ No messages returned, they're published instead
			},
		)
	}
	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
