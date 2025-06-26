package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	guildhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/watermill/handlers"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GuildRouter handles routing for guild module events.
type GuildRouter struct {
	logger     *slog.Logger
	Router     *message.Router
	subscriber message.Subscriber
	publisher  message.Publisher
}

// NewGuildRouter creates a new GuildRouter.
func NewGuildRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber message.Subscriber,
	publisher message.Publisher,
) *GuildRouter {
	return &GuildRouter{
		logger:     logger,
		Router:     router,
		subscriber: subscriber,
		publisher:  publisher,
	}
}

// Configure sets up the guild router with handlers.
func (r *GuildRouter) Configure(ctx context.Context, handlers *guildhandlers.GuildConfigHandler) error {
	eventsToHandlers := map[string]func(msg *message.Message) ([]*message.Message, error){
		guildevents.GuildSetupEventTopic:        handlers.HandleGuildSetup,
		guildevents.GuildConfigUpdateEventTopic: handlers.HandleGuildConfigUpdate,
		guildevents.GuildRemovedEventTopic:      handlers.HandleGuildRemoved,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("guild.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing guild message",
						attr.String("message_id", msg.UUID),
						attr.String("topic", topic),
						attr.Error(err),
					)
					return nil, err
				}

				return messages, nil
			},
		)
	}

	r.logger.InfoContext(ctx, "Guild router configured successfully")
	return nil
}
