package router

import (
	"context"
	"fmt"
	"log/slog"

	clubhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/club/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// ClubRouter handles routing for club module events.
type ClubRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// NewClubRouter creates a new ClubRouter.
func NewClubRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *ClubRouter {
	return &ClubRouter{
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

// Configure sets up the club router.
func (r *ClubRouter) Configure(ctx context.Context, handlers clubhandlers.Handlers) error {
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-club"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(ctx, handlers); err != nil {
		return fmt.Errorf("failed to register club handlers: %w", err)
	}

	return nil
}

func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-club." + topic

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"",
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

// RegisterHandlers registers club event handlers.
func (r *ClubRouter) RegisterHandlers(ctx context.Context, handlers clubhandlers.Handlers) error {
	var metrics handlerwrapper.ReturningMetrics

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	challengeTopics := []string{
		clubevents.ChallengeOpenedV1,
		clubevents.ChallengeAcceptedV1,
		clubevents.ChallengeDeclinedV1,
		clubevents.ChallengeWithdrawnV1,
		clubevents.ChallengeExpiredV1,
		clubevents.ChallengeHiddenV1,
		clubevents.ChallengeCompletedV1,
		clubevents.ChallengeRoundLinkedV1,
		clubevents.ChallengeRoundUnlinkedV1,
		clubevents.ChallengeRefreshedV1,
	}

	for _, topic := range challengeTopics {
		topic := topic
		registerHandler(deps, topic, func(ctx context.Context, payload *clubevents.ChallengeFactPayloadV1) ([]handlerwrapper.Result, error) {
			return handlers.HandleChallengeFact(ctx, topic, payload)
		})
	}

	r.logger.InfoContext(ctx, "Club router configured successfully")
	return nil
}

// Close stops the router.
func (r *ClubRouter) Close() error {
	return r.Router.Close()
}
