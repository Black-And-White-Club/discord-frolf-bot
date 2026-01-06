package leaderboardhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	leaderboarddiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/leaderboard/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Handlers defines the interface for leaderboard handlers.
type Handlers interface {
	// Leaderboard Updates
	HandleBatchTagAssigned(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardRetrieveRequest(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardData(msg *message.Message) ([]*message.Message, error)

	// Leaderboard Errors
	HandleLeaderboardUpdateFailed(msg *message.Message) ([]*message.Message, error)
	HandleLeaderboardRetrievalFailed(msg *message.Message) ([]*message.Message, error)

	// Tag Number Lookups
	HandleGetTagByDiscordID(msg *message.Message) ([]*message.Message, error)
	HandleGetTagByDiscordIDResponse(msg *message.Message) ([]*message.Message, error)

	// Tag Assignment
	HandleTagAssignRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagAssignedResponse(msg *message.Message) ([]*message.Message, error)
	HandleTagAssignFailedResponse(msg *message.Message) ([]*message.Message, error)

	// Tag Swap
	HandleTagSwapRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagSwappedResponse(msg *message.Message) ([]*message.Message, error)
	HandleTagSwapFailedResponse(msg *message.Message) ([]*message.Message, error)
}

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	Logger              *slog.Logger
	Config              *config.Config
	Helpers             utils.Helpers
	LeaderboardDiscord  leaderboarddiscord.LeaderboardDiscordInterface
	GuildConfigResolver guildconfig.GuildConfigResolver
	Tracer              trace.Tracer
	Metrics             discordmetrics.DiscordMetrics
	handlerWrapper      func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(
	logger *slog.Logger,
	config *config.Config,
	helpers utils.Helpers,
	leaderboardDiscord leaderboarddiscord.LeaderboardDiscordInterface,
	guildConfigResolver guildconfig.GuildConfigResolver,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) Handlers {
	return &LeaderboardHandlers{
		Logger:              logger,
		Config:              config,
		Helpers:             helpers,
		LeaderboardDiscord:  leaderboardDiscord,
		GuildConfigResolver: guildConfigResolver,
		Tracer:              tracer,
		Metrics:             metrics,
		handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
			return wrapHandler(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, helpers)
		},
	}
}

// wrapHandler wraps leaderboard handlers with tracing, logging, metrics, and error handling.
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

		// Attempt to unmarshal payload if struct provided
		if unmarshalTo != nil {
			if err := helpers.UnmarshalPayload(msg, unmarshalTo); err != nil {
				logger.ErrorContext(ctx, "Failed to unmarshal payload", attr.Error(err))
				metrics.RecordHandlerFailure(ctx, handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		// Execute actual handler logic
		result, err := handlerFunc(ctx, msg, unmarshalTo)
		if err != nil {
			logger.ErrorContext(ctx, "Handler returned error", attr.Error(err))
			metrics.RecordHandlerFailure(ctx, handlerName)
			return nil, err
		}

		logger.InfoContext(ctx, "Handler completed successfully")
		metrics.RecordHandlerSuccess(ctx, handlerName)

		return result, nil
	}
}
