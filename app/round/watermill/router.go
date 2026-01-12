package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/watermill/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewRoundRouter creates a new RoundRouter.
func NewRoundRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *RoundRouter {
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

// handlerDeps groups dependencies needed for handler registration
type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// registerHandler registers a pure transformation-pattern handler with typed payload
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "discord-round." + topic

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

// Configure sets up the router.
func (r *RoundRouter) Configure(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "RoundRouter.Configure called")
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("discord-round"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	err := r.RegisterHandlers(ctx, handlers)
	if err != nil {
		r.logger.ErrorContext(ctx, "RoundRouter.RegisterHandlers failed", attr.Error(err))
		return fmt.Errorf("failed to register round handlers: %w", err)
	}
	r.logger.InfoContext(ctx, "RoundRouter.Configure completed successfully")
	return nil
}

// RegisterHandlers registers all event handlers using the pure transformation pattern.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "RoundRouter.RegisterHandlers called")

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

	// Creation flow
	registerHandler(deps, sharedroundevents.RoundCreateModalSubmittedV1, handlers.HandleRoundCreateRequested)
	registerHandler(deps, roundevents.RoundCreatedV1, handlers.HandleRoundCreated)
	registerHandler(deps, roundevents.RoundCreationFailedV1, handlers.HandleRoundCreationFailed)
	registerHandler(deps, roundevents.RoundValidationFailedV1, handlers.HandleRoundValidationFailed)

	// Update flow
	registerHandler(deps, sharedroundevents.RoundUpdateModalSubmittedV1, handlers.HandleRoundUpdateRequested)
	registerHandler(deps, roundevents.RoundUpdatedV1, handlers.HandleRoundUpdated)
	registerHandler(deps, roundevents.RoundUpdateErrorV1, handlers.HandleRoundUpdateFailed)

	// Participation
	registerHandler(deps, sharedroundevents.RoundParticipantJoinRequestDiscordV1, handlers.HandleRoundParticipantJoinRequest)
	registerHandler(deps, roundevents.RoundParticipantRemovedV1, handlers.HandleRoundParticipantRemoved)

	// Scoring
	registerHandler(deps, roundevents.RoundParticipantScoreUpdatedV1, handlers.HandleParticipantScoreUpdated)
	registerHandler(deps, roundevents.RoundScoreUpdateErrorV1, handlers.HandleScoreUpdateError)

	// Score override bridging (CorrectScore service)
	registerHandler(deps, scoreevents.ScoreUpdatedV1, handlers.HandleScoreOverrideSuccess)
	// NOTE: We intentionally do NOT map ScoreBulkUpdateSuccess to per-user handler.
	// The per-user success events (score.update.success) are expected to be emitted individually
	// for each updated participant. The aggregate bulk success event lacks a specific user/score
	// and was causing empty participant payloads & failed embed updates when bridged.

	// Scorecard import flow
	registerHandler(deps, roundevents.ScorecardUploadedV1, handlers.HandleScorecardUploaded)
	registerHandler(deps, roundevents.ScorecardParseFailedV1, handlers.HandleScorecardParseFailed)
	registerHandler(deps, roundevents.ImportFailedV1, handlers.HandleImportFailed)
	registerHandler(deps, roundevents.ScorecardURLRequestedV1, handlers.HandleScorecardURLRequested)

	// Deletion flow
	registerHandler(deps, sharedroundevents.RoundDeleteRequestDiscordV1, handlers.HandleRoundDeleteRequested)

	// Lifecycle
	registerHandler(deps, roundevents.RoundDeletedV1, handlers.HandleRoundDeleted)
	registerHandler(deps, roundevents.RoundFinalizedDiscordV1, handlers.HandleRoundFinalized)
	registerHandler(deps, roundevents.RoundStartedV1, handlers.HandleRoundStarted)

	// Tag handling
	registerHandler(deps, roundevents.RoundParticipantJoinedV1, handlers.HandleRoundParticipantJoined)

	// Reminders
	registerHandler(deps, roundevents.RoundReminderSentV1, handlers.HandleRoundReminder)
	registerHandler(deps, roundevents.TagsUpdatedForScheduledRoundsV1, handlers.HandleTagsUpdatedForScheduledRounds)

	// NOTE: Several backend-only failure topics are intentionally NOT mapped to
	// Discord-side handlers. These topics are published by the `frolf-bot` backend
	// for observability/monitoring and do not require user-facing Discord actions.
	// Examples: `round.schedule.failed.v1`, `round.start.failed.v1`,
	// `round.finalization.error.v1`.
	// If an operational alert is needed in the future, add a dedicated handler
	// under `app/round/watermill/handlers/round_failures.go` and register it here.

	r.logger.InfoContext(ctx, "RoundRouter.RegisterHandlers completed successfully")
	return nil
}

// Close stops the router.
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
