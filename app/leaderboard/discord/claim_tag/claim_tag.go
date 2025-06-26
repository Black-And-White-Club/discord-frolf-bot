package claimtag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type ClaimTagManager interface {
	HandleClaimTagCommand(ctx context.Context, i *discordgo.InteractionCreate) (ClaimTagOperationResult, error)
	UpdateInteractionResponse(ctx context.Context, correlationID, message string) (ClaimTagOperationResult, error) // Add this method
}

type claimTagManager struct {
	session          discord.Session
	eventBus         eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (ClaimTagOperationResult, error)) (ClaimTagOperationResult, error)
}

type ClaimTagOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

func NewClaimTagManager(
	session discord.Session,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) ClaimTagManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating ClaimTagManager")
	}
	return &claimTagManager{
		session:          session,
		eventBus:         eventBus,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ClaimTagOperationResult, error)) (ClaimTagOperationResult, error) {
			return wrapClaimTagOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

func wrapClaimTagOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (ClaimTagOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (result ClaimTagOperationResult, err error) { // Named returns
	if fn == nil {
		return ClaimTagOperationResult{Error: errors.New("operation function is nil")}, nil
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
			recoveredErr := fmt.Errorf("panic in %s: %v", operationName, r)
			span.RecordError(recoveredErr)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(recoveredErr))
			}
			if metrics != nil {
				metrics.RecordAPIError(ctx, operationName, "panic")
			}
			// Set named return values
			result = ClaimTagOperationResult{Error: recoveredErr}
			err = recoveredErr
		}
	}()

	result, err = fn(ctx)
	if err != nil {
		wrapped := fmt.Errorf("%s operation error: %w", operationName, err)
		span.RecordError(wrapped)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped))
		}
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return ClaimTagOperationResult{Error: wrapped}, wrapped
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
