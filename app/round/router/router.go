package router

import (
	"context"
	"fmt"
	"log/slog"

	handlers "github.com/Black-And-White-Club/discord-frolf-bot/app/round/handlers"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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
	nativeSubscriber message.Subscriber // separate consumer group for fan-out handlers
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
	nativeSubscriber message.Subscriber,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *RoundRouter {
	return &RoundRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		nativeSubscriber: nativeSubscriber,
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
	suffix ...string,
) {
	handlerName := "discord-round." + topic
	if len(suffix) > 0 {
		handlerName += suffix[0]
	}

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

// registerHandlerWithSub registers a handler using an explicit subscriber.
// Used for fan-out patterns where multiple handlers need independent consumer
// groups on the same topic.
func registerHandlerWithSub[T any](
	deps handlerDeps,
	sub message.Subscriber,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
	suffix ...string,
) {
	handlerName := "discord-round." + topic
	if len(suffix) > 0 {
		handlerName += suffix[0]
	}

	deps.router.AddHandler(
		handlerName,
		topic,
		sub,
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

// Configure sets up the router.
func (r *RoundRouter) Configure(ctx context.Context, handlers handlers.Handlers) error {
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
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers handlers.Handlers) error {
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
	registerHandler(deps, discordroundevents.RoundCreateModalSubmittedV1, handlers.HandleRoundCreateRequested)
	registerHandler(deps, roundevents.RoundCreatedV1, handlers.HandleRoundCreated)

	// Native Event Creation (parallel to embed creation, independent consumer group).
	// Uses a separate subscriber with its own NATS consumer so both this handler
	// and the embed handler above each receive every RoundCreatedV1 message (fan-out).
	registerHandlerWithSub(deps, r.nativeSubscriber, roundevents.RoundCreatedV1, handlers.HandleRoundCreatedForNativeEvent, ".native")

	registerHandler(deps, roundevents.RoundCreationFailedV1, handlers.HandleRoundCreationFailed)
	registerHandler(deps, roundevents.RoundValidationFailedV1, handlers.HandleRoundValidationFailed)

	// Update flow
	registerHandler(deps, discordroundevents.RoundUpdateModalSubmittedV1, handlers.HandleRoundUpdateRequested)
	registerHandler(deps, roundevents.RoundUpdatedV1, handlers.HandleRoundUpdated)
	registerHandler(deps, roundevents.RoundUpdateErrorV1, handlers.HandleRoundUpdateFailed)

	// Participation
	registerHandler(deps, discordroundevents.RoundParticipantJoinRequestDiscordV1, handlers.HandleRoundParticipantJoinRequest)
	registerHandler(deps, roundevents.RoundParticipantRemovedV1, handlers.HandleRoundParticipantRemoved)

	// Scoring
	// Discord->Round bridge: translate discord-scoped round score submissions
	// into the canonical round domain topic so the round handlers can process
	// first-time submissions and participant creation.
	registerHandler(deps, discordroundevents.RoundScoreUpdateRequestDiscordV1, handlers.HandleDiscordRoundScoreUpdate)
	registerHandler(deps, roundevents.RoundParticipantScoreUpdatedV1, handlers.HandleParticipantScoreUpdated)
	registerHandler(deps, roundevents.RoundScoresBulkUpdatedV1, handlers.HandleScoresBulkUpdated)
	registerHandler(deps, roundevents.RoundScoreUpdateErrorV1, handlers.HandleScoreUpdateError)

	// Score override bridging (CorrectScore service)
	registerHandler(deps, sharedevents.ScoreUpdatedV1, handlers.HandleScoreOverrideSuccess)
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
	registerHandler(deps, discordroundevents.RoundDeleteRequestDiscordV1, handlers.HandleRoundDeleteRequested)

	// Lifecycle
	registerHandler(deps, roundevents.RoundDeletedV1, handlers.HandleRoundDeleted)
	registerHandler(deps, roundevents.RoundFinalizedDiscordV1, handlers.HandleRoundFinalized)
	registerHandler(deps, roundevents.RoundStartedDiscordV1, handlers.HandleRoundStarted)

	// Tag handling
	registerHandler(deps, roundevents.RoundParticipantJoinedV1, handlers.HandleRoundParticipantJoined)
	registerHandler(deps, roundevents.RoundParticipantsUpdatedV1, handlers.HandleRoundParticipantsUpdated)

	// Reminders
	registerHandler(deps, roundevents.RoundReminderSentV1, handlers.HandleRoundReminder)
	registerHandler(deps, roundevents.ScheduledRoundsSyncedV1, handlers.HandleScheduledRoundsSynced)

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

// Close stops the router and its subscriber resources.
func (r *RoundRouter) Close() error {
	err := r.Router.Close()
	if r.nativeSubscriber != nil {
		if closeErr := r.nativeSubscriber.Close(); closeErr != nil {
			r.logger.Error("Failed to close native subscriber", "error", closeErr)
		}
	}
	return err
}
