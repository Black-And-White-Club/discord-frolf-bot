package roundreminder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// RoundReminderManager defines the interface for round reminder operations.
type RoundReminderManager interface {
	SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (RoundReminderOperationResult, error)
}
type roundReminderManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (RoundReminderOperationResult, error)) (RoundReminderOperationResult, error)
	guildConfigResolver guildconfig.GuildConfigResolver
}

// NewRoundReminderManager creates a new RoundReminderManager instance.
func NewRoundReminderManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) RoundReminderManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating RoundReminderManager")
	}
	return &roundReminderManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (RoundReminderOperationResult, error)) (RoundReminderOperationResult, error) {
			return wrapRoundReminderOperation(ctx, opName, fn, logger, tracer, metrics)
		},
		guildConfigResolver: guildConfigResolver,
	}
}

// wrapRoundReminderOperation adds tracing, metrics, and error handling for operations.
func wrapRoundReminderOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (RoundReminderOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (RoundReminderOperationResult, error) {
	if fn == nil {
		return RoundReminderOperationResult{Error: errors.New("operation function is nil")}, nil
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	start := time.Now()

	// Create local reference to prevent data race
	localMetrics := metrics

	// First defer: record duration metrics
	defer func() {
		if localMetrics != nil {
			duration := time.Since(start)
			localMetrics.RecordAPIRequestDuration(ctx, operationName, duration)
		}
	}()

	// Second defer: handle panics
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in %s: %v", operationName, r)
			span.RecordError(err)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(err))
			}
			if localMetrics != nil {
				localMetrics.RecordAPIError(ctx, operationName, "panic")
			}
		}
	}()

	// Execute the operation function
	result, err := fn(ctx)
	if err != nil {
		wrapped := fmt.Errorf("%s operation error: %w", operationName, err)
		span.RecordError(wrapped)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped))
		}
		if localMetrics != nil {
			localMetrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return RoundReminderOperationResult{Error: wrapped}, wrapped
	}

	if result.Error != nil {
		span.RecordError(result.Error)
		if localMetrics != nil {
			localMetrics.RecordAPIError(ctx, operationName, "result_error")
		}
	} else if localMetrics != nil {
		localMetrics.RecordAPIRequest(ctx, operationName)
	}

	return result, nil
}

// RoundReminderOperationResult is the standard result container for operations.
type RoundReminderOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
