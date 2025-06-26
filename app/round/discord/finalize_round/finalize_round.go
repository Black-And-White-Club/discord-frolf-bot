package finalizeround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// FinalizeRoundManager defines the interface for finalize round operations.
type FinalizeRoundManager interface {
	TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error)
	FinalizeScorecardEmbed(ctx context.Context, eventMessageID string, channelID string, embedPayload roundevents.RoundFinalizedEmbedUpdatePayload) (FinalizeRoundOperationResult, error)
}

type finalizeRoundManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (FinalizeRoundOperationResult, error)) (FinalizeRoundOperationResult, error)
}

// NewFinalizeRoundManager creates a new FinalizeRoundManager instance.
func NewFinalizeRoundManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) FinalizeRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating FinalizeRoundManager")
	}
	return &finalizeRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (FinalizeRoundOperationResult, error)) (FinalizeRoundOperationResult, error) {
			return wrapFinalizeRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

func wrapFinalizeRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (FinalizeRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (FinalizeRoundOperationResult, error) {
	if fn == nil {
		return FinalizeRoundOperationResult{Error: errors.New("operation function is nil")}, nil
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	start := time.Now()
	defer func() {
		if metrics != nil {
			duration := time.Since(start)
			metrics.RecordAPIRequestDuration(ctx, operationName, duration)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in %s: %v", operationName, r)
			span.RecordError(err)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(err))
			}
			if metrics != nil {
				metrics.RecordAPIError(ctx, operationName, "panic")
			}
		}
	}()

	result, err := fn(ctx)
	if err != nil {
		wrapped := fmt.Errorf("%s operation error: %w", operationName, err)
		span.RecordError(wrapped)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped))
		}
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return FinalizeRoundOperationResult{Error: wrapped}, wrapped
	}

	if result.Error != nil {
		span.RecordError(result.Error)
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "result_error")
		}
	} else if metrics != nil {
		metrics.RecordAPIRequest(ctx, operationName)
	}

	return result, nil
}

type FinalizeRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
