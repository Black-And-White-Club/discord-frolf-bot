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
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewRoundRouter creates a new RoundRouter.

func NewRoundRouter(logger *slog.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, config *config.Config, helper utils.Helpers, tracer observability.Tracer) *RoundRouter {
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
func (r *RoundRouter) Configure(handlers roundhandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.Retry{MaxRetries: 3}.Middleware,
		r.middlewareHelper.CommonMetadataMiddleware("discord-round"),
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
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		discordroundevents.RoundCreateModalSubmit:       handlers.HandleRoundCreateRequested,
		discordroundevents.RoundCreatedTopic:            handlers.HandleRoundCreated,
		discordroundevents.RoundStartedTopic:            handlers.HandleRoundStarted,
		discordroundevents.RoundParticipantJoinReqTopic: handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantJoined:              handlers.HandleRoundParticipantJoined,
		discordroundevents.RoundValidationFailed:        handlers.HandleRoundValidationFailed,
		// discordroundevents.RoundUpdateRequestTopic:           handlers.HandleRoundUpdateRequest,
		// discordroundevents.RoundUpdatedTopic:                 handlers.HandleRoundUpdated,
		discordroundevents.RoundDeletedTopic: handlers.HandleRoundDeleted,
		roundevents.DiscordRoundFinalized:    handlers.HandleRoundFinalized,
		// discordroundevents.RoundReminderTopic:                handlers.HandleRoundReminder,
		// discordroundevents.RoundScoreUpdateRequestTopic:      handlers.HandleRoundScoreUpdateRequest,
		roundevents.RoundParticipantScoreUpdated: handlers.HandleParticipantScoreUpdated,
		roundevents.RoundScoreUpdateError:        handlers.HandleScoreUpdateError,
		roundevents.RoundCreationFailed:          handlers.HandleRoundCreationFailed,
	}
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-round.%s", topic)
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
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
