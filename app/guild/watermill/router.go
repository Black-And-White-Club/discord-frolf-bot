package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// GuildRouter handles routing for guild module events.
type GuildRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewGuildRouter creates a new GuildRouter.
func NewGuildRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *GuildRouter {
	return &GuildRouter{
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
func (r *GuildRouter) Configure(ctx context.Context, handlers guildhandlers.Handlers) error {
	// Note: Using Discord metrics instead of separate Prometheus metrics to avoid conflicts

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-guild"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register guild handlers: %w", err)
	}
	return nil
}

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// This centralizes routing logic in the router (not in handlers or helpers).
func (r *GuildRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Extract base topic from handlerName format: "discord-guild.{topic}"
	// Map handler input topic → output topic(s)

	switch {
	case handlerName == "discord-guild."+guildevents.GuildConfigCreatedV1:
		// HandleGuildConfigCreated doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigCreationFailedV1:
		// HandleGuildConfigCreationFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigUpdatedV1:
		// HandleGuildConfigUpdated doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigUpdateFailedV1:
		// HandleGuildConfigUpdateFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigRetrievedV1:
		// HandleGuildConfigRetrieved doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigRetrievalFailedV1:
		// HandleGuildConfigRetrievalFailed doesn't return messages (nil)
		return ""

	case handlerName == "discord-guild."+guildevents.GuildConfigDeletedV1:
		// HandleGuildConfigDeleted always returns GuildConfigDeletionResultsV1
		return guildevents.GuildConfigDeletionResultsV1

	case handlerName == "discord-guild."+guildevents.GuildConfigDeletionFailedV1:
		// HandleGuildConfigDeletionFailed doesn't return messages (nil)
		return ""

	default:
		r.logger.Warn("unknown handler in topic resolution",
			attr.String("handler", handlerName),
		)
		// Fallback to metadata (graceful degradation during migration)
		return msg.Metadata.Get("topic")
	}
}

// RegisterHandlers registers event handlers.
func (r *GuildRouter) RegisterHandlers(ctx context.Context, handlers guildhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Guild Handlers")

	eventsToHandlers := map[string]message.HandlerFunc{
		// Guild config creation/setup flow - affects command registration
		guildevents.GuildConfigCreatedV1:        handlers.HandleGuildConfigCreated,
		guildevents.GuildConfigCreationFailedV1: handlers.HandleGuildConfigCreationFailed,

		// Guild config update flow - may affect command permissions
		guildevents.GuildConfigUpdatedV1:      handlers.HandleGuildConfigUpdated,
		guildevents.GuildConfigUpdateFailedV1: handlers.HandleGuildConfigUpdateFailed,

		// Guild config retrieval flow - informational only, no command action
		guildevents.GuildConfigRetrievedV1:       handlers.HandleGuildConfigRetrieved,
		guildevents.GuildConfigRetrievalFailedV1: handlers.HandleGuildConfigRetrievalFailed,

		// Guild config deletion flow - affects command registration
		guildevents.GuildConfigDeletedV1:        handlers.HandleGuildConfigDeleted,
		guildevents.GuildConfigDeletionFailedV1: handlers.HandleGuildConfigDeletionFailed,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-guild.%s", topic)

		// Use environment-specific queue groups for multi-tenant scalability
		// This ensures only one instance processes each message per environment
		queueGroup := fmt.Sprintf("guild-handlers-%s", r.config.Observability.Environment)

		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			queueGroup,
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing guild message",
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

	r.logger.InfoContext(ctx, "Guild router configured successfully",
		attr.Int("registered_handlers", len(eventsToHandlers)))

	return nil
}

// Close gracefully shuts down the router
func (r *GuildRouter) Close() error {
	if r.Router != nil {
		return r.Router.Close()
	}
	return nil
}
