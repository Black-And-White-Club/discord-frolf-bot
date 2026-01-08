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

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// This centralizes routing logic in the router (not in handlers or helpers).
func (r *RoundRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Map handler input topic → output topic(s)
	// Based on analysis of all 23 handlers in handlers/ directory

	switch {
	// Creation flow
	case handlerName == "discord-round."+sharedroundevents.RoundCreateModalSubmittedV1:
		// HandleRoundCreateRequested always returns RoundCreationRequestedV1
		return roundevents.RoundCreationRequestedV1

	case handlerName == "discord-round."+roundevents.RoundCreatedV1:
		// HandleRoundCreated always returns RoundEventMessageIDUpdateV1
		return roundevents.RoundEventMessageIDUpdateV1

	case handlerName == "discord-round."+roundevents.RoundCreationFailedV1:
		// HandleRoundCreationFailed returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.RoundValidationFailedV1:
		// HandleRoundValidationFailed returns nil (Discord API only)
		return ""

	// Update flow
	case handlerName == "discord-round."+sharedroundevents.RoundUpdateModalSubmittedV1:
		// HandleRoundUpdateRequested always returns RoundUpdateRequestedV1
		return roundevents.RoundUpdateRequestedV1

	case handlerName == "discord-round."+roundevents.RoundUpdatedV1:
		// HandleRoundUpdated returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.RoundUpdateErrorV1:
		// HandleRoundUpdateFailed returns nil (logs error only)
		return ""

	// Participation
	case handlerName == "discord-round."+sharedroundevents.RoundParticipantJoinRequestDiscordV1:
		// HandleRoundParticipantJoinRequest always returns RoundParticipantJoinRequestedV1
		return roundevents.RoundParticipantJoinRequestedV1

	case handlerName == "discord-round."+roundevents.RoundParticipantRemovedV1:
		// HandleRoundParticipantRemoved returns nil (Discord API only)
		return ""

	// Scoring
	case handlerName == "discord-round."+roundevents.RoundParticipantScoreUpdatedV1:
		// HandleParticipantScoreUpdated returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.RoundScoreUpdateErrorV1:
		// HandleScoreUpdateError always returns RoundTraceEventV1
		return roundevents.RoundTraceEventV1

	// Score override bridging (CorrectScore service)
	case handlerName == "discord-round."+scoreevents.ScoreUpdatedV1:
		// HandleScoreOverrideSuccess bridges to RoundParticipantScoreUpdatedV1
		return roundevents.RoundParticipantScoreUpdatedV1

	// Scorecard import flow
	case handlerName == "discord-round."+roundevents.ScorecardUploadedV1:
		// HandleScorecardUploaded returns nil (validation only)
		return ""

	case handlerName == "discord-round."+roundevents.ScorecardParseFailedV1:
		// HandleScorecardParseFailed returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.ImportFailedV1:
		// HandleImportFailed returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.ScorecardURLRequestedV1:
		// HandleScorecardURLRequested returns nil (TODO: future Discord API)
		return ""

	// Deletion flow
	case handlerName == "discord-round."+sharedroundevents.RoundDeleteRequestDiscordV1:
		// HandleRoundDeleteRequested always returns RoundDeleteRequestedV1
		return roundevents.RoundDeleteRequestedV1

	// Lifecycle
	case handlerName == "discord-round."+roundevents.RoundDeletedV1:
		// HandleRoundDeleted always returns RoundTraceEventV1
		return roundevents.RoundTraceEventV1

	case handlerName == "discord-round."+roundevents.RoundFinalizedDiscordV1:
		// HandleRoundFinalized always returns RoundTraceEventV1
		return roundevents.RoundTraceEventV1

	case handlerName == "discord-round."+roundevents.RoundStartedV1:
		// HandleRoundStarted always returns RoundTraceEventV1
		return roundevents.RoundTraceEventV1

	// Tag handling
	case handlerName == "discord-round."+roundevents.RoundParticipantJoinedV1:
		// HandleRoundParticipantJoined returns nil (Discord API only)
		return ""

	// Reminders
	case handlerName == "discord-round."+roundevents.RoundReminderSentV1:
		// HandleRoundReminder returns nil (Discord API only)
		return ""

	case handlerName == "discord-round."+roundevents.TagsUpdatedForScheduledRoundsV1:
		// HandleTagsUpdatedForScheduledRounds returns nil (Discord API only)
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

					r.logger.InfoContext(msgCtx, "Publishing message",
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
	r.logger.InfoContext(ctx, "RoundRouter.RegisterHandlers completed successfully")
	return nil
}

// Close stops the router.
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
