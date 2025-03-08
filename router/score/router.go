package scorerouter

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/score"
	scorehandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	tempo "github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// ScoreRouter handles routing for score module events.
type ScoreRouter struct {
	logger     observability.Logger
	Router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	session    discord.Session
	tracer     tempo.Tracer
}

// NewScoreRouter creates a new ScoreRouter.

func NewScoreRouter(logger observability.Logger, router *message.Router, subscriber eventbus.EventBus, publisher eventbus.EventBus, session discord.Session, tracer tempo.Tracer) *ScoreRouter {
	return &ScoreRouter{
		logger:     logger,
		Router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		session:    session,
		tracer:     tracer,
	}
}

// Configure sets up the router.
func (r *ScoreRouter) Configure(handlers scorehandlers.Handlers, eventbus eventbus.EventBus) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		middleware.Retry{
			MaxRetries: 3,
		}.Middleware,
		r.tracer.TraceHandler,
		r.LokiLoggingMiddleware,
	)
	if err := r.RegisterHandlers(context.Background(), handlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// LokiLoggingMiddleware is the custom Watermill middleware.
func (r *ScoreRouter) LokiLoggingMiddleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		startTime := time.Now()
		ctx := msg.Context()
		handlerName := msg.Metadata.Get("handler_name")
		domain := msg.Metadata.Get("domain")
		r.logger.Info(ctx, "Received message",
			attr.CorrelationIDFromMsg(msg),
			attr.Topic(msg.Metadata.Get("topic")),
			attr.MessageID(msg),
			attr.String("handler", handlerName),
			attr.String("domain", domain),
		)
		for key, value := range msg.Metadata {
			r.logger.Info(ctx, "Message metadata", attr.String(key, value))
		}
		producedMessages, err := next(msg)
		duration := time.Since(startTime)
		if err != nil {
			r.logger.Error(ctx, "Error processing message",
				attr.CorrelationIDFromMsg(msg),
				attr.Topic(msg.Metadata.Get("topic")),
				attr.MessageID(msg),
				attr.Duration("duration", duration),
				attr.String("handler", handlerName),
				attr.String("domain", domain),
				attr.Error(err),
			)
		} else {
			r.logger.Info(ctx, "Message processed successfully",
				attr.CorrelationIDFromMsg(msg),
				attr.Topic(msg.Metadata.Get("topic")),
				attr.MessageID(msg),
				attr.Duration("duration", duration),
				attr.String("handler", handlerName),
				attr.String("domain", domain),
			)
		}
		return producedMessages, err
	}
}

// RegisterHandlers registers event handlers.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		discorduserevents.ScoreUpdateRequestTopic:  handlers.HandleScoreUpdateRequest,
		discorduserevents.ScoreUpdateResponseTopic: handlers.HandleScoreUpdateResponse,
	}
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord.score.%s", topic)
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
func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
