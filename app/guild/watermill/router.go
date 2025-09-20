package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	discordguildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
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

// RegisterHandlers registers event handlers.
func (r *GuildRouter) RegisterHandlers(ctx context.Context, handlers guildhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Guild Handlers")

	eventsToHandlers := map[string]message.HandlerFunc{
		// Initial guild setup request from Discord
		discordguildevents.GuildSetupEventTopic: handlers.HandleGuildSetupRequest,

		// Guild config creation/setup flow - affects command registration
		guildevents.GuildConfigCreated:        handlers.HandleGuildConfigCreated,
		guildevents.GuildConfigCreationFailed: handlers.HandleGuildConfigCreationFailed,

		// Guild config update flow - may affect command permissions
		guildevents.GuildConfigUpdated:      handlers.HandleGuildConfigUpdated,
		guildevents.GuildConfigUpdateFailed: handlers.HandleGuildConfigUpdateFailed,

		// Guild config retrieval flow - informational only, no command action
		guildevents.GuildConfigRetrieved:       handlers.HandleGuildConfigRetrieved,
		guildevents.GuildConfigRetrievalFailed: handlers.HandleGuildConfigRetrievalFailed,

		// Guild config deletion flow - affects command registration
		guildevents.GuildConfigDeleted:        handlers.HandleGuildConfigDeleted,
		guildevents.GuildConfigDeletionFailed: handlers.HandleGuildConfigDeletionFailed,
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
