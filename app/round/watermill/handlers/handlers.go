package roundhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	rounddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Handlers defines the interface for round handlers.
type Handlers interface {
	HandleRoundStarted(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateError(msg *message.Message) ([]*message.Message, error)
	HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundReminder(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoinRequest(msg *message.Message) ([]*message.Message, error)
	HandleRoundParticipantJoined(msg *message.Message) ([]*message.Message, error)
	HandleRoundFinalized(msg *message.Message) ([]*message.Message, error)
	HandleRoundDeleted(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreateRequested(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreated(msg *message.Message) ([]*message.Message, error)
	HandleRoundValidationFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoundCreationFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateRequested(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoundUpdateValidationFailed(msg *message.Message) ([]*message.Message, error)
}

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	Logger         *slog.Logger
	Config         *config.Config
	Helpers        utils.Helpers
	RoundDiscord   rounddiscord.RoundDiscordInterface
	Tracer         trace.Tracer
	Metrics        discordmetrics.DiscordMetrics
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	roundDiscord rounddiscord.RoundDiscordInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) Handlers {
	return &RoundHandlers{
		Logger:       logger,
		Config:       config,
		Helpers:      helpers,
		RoundDiscord: roundDiscord,
		Tracer:       tracer,
		Metrics:      metrics,
		handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
			return wrapHandler(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, helpers)
		},
	}
}

// wrapHandler is the handler wrapper for common logging, tracing, and metrics.
func wrapHandler(
	handlerName string,
	unmarshalTo interface{},
	handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error),
	logger *slog.Logger,
	metrics discordmetrics.DiscordMetrics,
	tracer trace.Tracer,
	helpers utils.Helpers,
) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx, span := tracer.Start(
			msg.Context(), handlerName,
			trace.WithAttributes(
				attribute.String("message.id", msg.UUID),
				attribute.String("message.correlation_id", middleware.MessageCorrelationID(msg)),
			),
		)
		defer span.End()

		start := time.Now()
		metrics.RecordHandlerAttempt(ctx, handlerName)

		defer func() {
			duration := time.Since(start)
			metrics.RecordHandlerDuration(ctx, handlerName, duration)
		}()

		logger := logger.With(
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
		)

		logger.InfoContext(ctx, "Handler started", slog.String("handler", handlerName))

		// Unmarshal only if target struct provided
		if unmarshalTo != nil {
			if err := helpers.UnmarshalPayload(msg, unmarshalTo); err != nil {
				logger.ErrorContext(ctx, "Failed to unmarshal payload",
					attr.Error(err),
				)
				metrics.RecordHandlerFailure(ctx, handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		// Run the handler
		result, err := handlerFunc(ctx, msg, unmarshalTo)
		if err != nil {
			logger.ErrorContext(ctx, "Handler returned error",
				attr.Error(err),
			)
			metrics.RecordHandlerFailure(ctx, handlerName)
			return nil, err
		}

		logger.InfoContext(ctx, "Handler completed successfully")
		metrics.RecordHandlerSuccess(ctx, handlerName)

		return result, nil
	}
}
