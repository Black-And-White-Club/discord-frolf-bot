package scorehandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Handler defines the interface for score-related Watermill event handlers.
type Handler interface {
	HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateSuccess(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateFailure(msg *message.Message) ([]*message.Message, error)
	HandleProcessRoundScoresFailed(msg *message.Message) ([]*message.Message, error)
}

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	Logger         *slog.Logger
	Config         *config.Config
	Session        discord.Session
	Helper         utils.Helpers
	Tracer         trace.Tracer
	Metrics        discordmetrics.DiscordMetrics
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewScoreHandlers creates a new ScoreHandlers struct.
func NewScoreHandlers(
	logger *slog.Logger,
	config *config.Config,
	session discord.Session,
	helpers utils.Helpers,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) Handler {
	return &ScoreHandlers{
		Logger:  logger,
		Config:  config,
		Session: session,
		Helper:  helpers,
		Tracer:  tracer,
		Metrics: metrics,
		handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
			return wrapHandler(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, helpers)
		},
	}
}

// wrapHandler wraps the message handler with observability and error handling.
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
		ctx, span := tracer.Start(msg.Context(), handlerName, trace.WithAttributes(
			attribute.String("message.id", msg.UUID),
			attribute.String("message.correlation_id", middleware.MessageCorrelationID(msg)),
		))
		defer span.End()

		metrics.RecordHandlerAttempt(ctx, handlerName)

		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime).Seconds()
			metrics.RecordHandlerDuration(ctx, handlerName, time.Duration(duration))
		}()

		logger.InfoContext(ctx, handlerName+" triggered",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
		)

		payloadInstance := unmarshalTo

		if payloadInstance != nil {
			if err := helpers.UnmarshalPayload(msg, payloadInstance); err != nil {
				logger.ErrorContext(ctx, "Failed to unmarshal payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				metrics.RecordHandlerFailure(ctx, handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		result, err := handlerFunc(ctx, msg, payloadInstance)
		if err != nil {
			logger.ErrorContext(ctx, "Error in "+handlerName,
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err),
			)
			metrics.RecordHandlerFailure(ctx, handlerName)
			return nil, err
		}

		logger.InfoContext(ctx, handlerName+" completed successfully", attr.CorrelationIDFromMsg(msg))
		metrics.RecordHandlerSuccess(ctx, handlerName)

		return result, nil
	}
}
