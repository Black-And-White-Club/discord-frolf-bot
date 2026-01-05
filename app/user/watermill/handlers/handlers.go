// app/user/watermill/handlers/user/handlers.go
package userhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserHandler defines the interface for user-related Watermill event handlers.
type Handler interface {
	HandleRoleUpdateCommand(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateButtonPress(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdated(msg *message.Message) ([]*message.Message, error)
	HandleRoleUpdateFailed(msg *message.Message) ([]*message.Message, error)
	HandleUserCreated(msg *message.Message) ([]*message.Message, error)
	HandleUserCreationFailed(msg *message.Message) ([]*message.Message, error)
	HandleRoleAdded(msg *message.Message) ([]*message.Message, error)
	HandleRoleAdditionFailed(msg *message.Message) ([]*message.Message, error)
	HandleAddRole(msg *message.Message) ([]*message.Message, error)
}

// userHandlers handles user-related events.
type UserHandlers struct {
	Logger         *slog.Logger
	Config         *config.Config
	Helper         utils.Helpers
	UserDiscord    userdiscord.UserDiscordInterface
	Tracer         trace.Tracer
	Metrics        discordmetrics.DiscordMetrics
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewUserHandlers creates a new UserHandlers struct.
func NewUserHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	userDiscord userdiscord.UserDiscordInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) Handler {
	return &UserHandlers{
		Logger:      logger,
		Config:      config,
		Helper:      helpers,
		UserDiscord: userDiscord,
		Tracer:      tracer,
		Metrics:     metrics,
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
		// Start a span for tracing
		ctx, span := tracer.Start(msg.Context(), handlerName, trace.WithAttributes(
			attribute.String("message.id", msg.UUID),
			attribute.String("message.correlation_id", middleware.MessageCorrelationID(msg)),
		))
		defer span.End()

		// Record metrics for handler attempt
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

		// Create a new instance of the payload type
		payloadInstance := unmarshalTo

		// Unmarshal payload if a target is provided
		if payloadInstance != nil {
			if err := helpers.UnmarshalPayload(msg, payloadInstance); err != nil {
				logger.ErrorContext(ctx, "Failed to unmarshal payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err))
				metrics.RecordHandlerFailure(ctx, handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		// Call the actual handler logic
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
