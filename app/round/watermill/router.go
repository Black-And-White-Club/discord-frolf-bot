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

// RegisterHandlers registers event handlers.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "RoundRouter.RegisterHandlers called")

	eventsToHandlers := map[string]message.HandlerFunc{
		// Creation flow
		sharedroundevents.RoundCreateModalSubmittedV1: handlers.HandleRoundCreateRequested,
		roundevents.RoundCreatedV1:                    handlers.HandleRoundCreated,
		roundevents.RoundCreationFailedV1:             handlers.HandleRoundCreationFailed,
		roundevents.RoundValidationFailedV1:           handlers.HandleRoundValidationFailed,

		// Update flow
		sharedroundevents.RoundUpdateModalSubmittedV1: handlers.HandleRoundUpdateRequested,
		roundevents.RoundUpdatedV1:                    handlers.HandleRoundUpdated,
		roundevents.RoundUpdateErrorV1:                handlers.HandleRoundUpdateFailed,

		// Participation
		sharedroundevents.RoundParticipantJoinRequestDiscordV1: handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantRemovedV1:                  handlers.HandleRoundParticipantRemoved,

		// Scoring
		roundevents.RoundParticipantScoreUpdatedV1: handlers.HandleParticipantScoreUpdated,
		roundevents.RoundScoreUpdateErrorV1:        handlers.HandleScoreUpdateError,

		// Score override bridging (CorrectScore service)
		scoreevents.ScoreUpdatedV1: handlers.HandleScoreOverrideSuccess,
		// NOTE: We intentionally do NOT map ScoreBulkUpdateSuccess to per-user handler.
		// The per-user success events (score.update.success) are expected to be emitted individually
		// for each updated participant. The aggregate bulk success event lacks a specific user/score
		// and was causing empty participant payloads & failed embed updates when bridged.

		// Scorecard import flow
		roundevents.ScorecardUploadedV1:     handlers.HandleScorecardUploaded,
		roundevents.ScorecardParseFailedV1:  handlers.HandleScorecardParseFailed,
		roundevents.ImportFailedV1:          handlers.HandleImportFailed,
		roundevents.ScorecardURLRequestedV1: handlers.HandleScorecardURLRequested,

		// Deletion flow
		sharedroundevents.RoundDeleteRequestDiscordV1: handlers.HandleRoundDeleteRequested,

		// Lifecycle
		roundevents.RoundDeletedV1:          handlers.HandleRoundDeleted,
		roundevents.RoundFinalizedDiscordV1: handlers.HandleRoundFinalized,
		roundevents.RoundStartedV1:          handlers.HandleRoundStarted,

		// Tag handling
		roundevents.RoundParticipantJoinedV1: handlers.HandleRoundParticipantJoined,

		// Reminders
		roundevents.RoundReminderSentV1: handlers.HandleRoundReminder,

		roundevents.TagsUpdatedForScheduledRoundsV1: handlers.HandleTagsUpdatedForScheduledRounds,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("discord-round.%s", topic)
		r.logger.InfoContext(ctx, "Registering handler for topic", attr.String("topic", topic), attr.String("handler", handlerName))
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				// Use message context, not captured registration context
				msgCtx := msg.Context()

				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(msgCtx, "Error processing message",
						attr.String("message_id", msg.UUID),
						attr.Error(err),
					)
					return nil, err
				}

				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						r.logger.InfoContext(msgCtx, "Publishing message",
							attr.String("message_id", m.UUID),
							attr.String("topic", publishTopic),
						)
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						r.logger.WarnContext(msgCtx, "Message missing topic metadata",
							attr.String("message_id", m.UUID),
						)
					}
				}
				return nil, nil
			},
		)
	}
	r.logger.InfoContext(ctx, "RoundRouter.RegisterHandlers completed successfully")
	return nil
}

// Close stops the router.
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
