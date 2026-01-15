package startround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
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

// StartRoundOperationResult defines the result structure for start round operations.
type StartRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// StartRoundManager defines the interface for create round operations.
type StartRoundManager interface {
	TransformRoundToScorecard(ctx context.Context, payload *roundevents.DiscordRoundStartPayloadV1, existingEmbed *discordgo.MessageEmbed) (StartRoundOperationResult, error)
	UpdateRoundToScorecard(ctx context.Context, channelID, messageID string, payload *roundevents.DiscordRoundStartPayloadV1) (StartRoundOperationResult, error)
}

// startRoundManager implements the StartRoundManager interface.
type startRoundManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error)
	guildConfigResolver guildconfig.GuildConfigResolver
}

// NewStartRoundManager creates a new StartRoundManager instance.
func NewStartRoundManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) StartRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating StartRoundManager")
	}
	return &startRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (StartRoundOperationResult, error)) (StartRoundOperationResult, error) {
			return wrapStartRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
		guildConfigResolver: guildConfigResolver,
		guildConfigCache:    guildConfigCache,
		interactionStore:    interactionStore,
	}
}

// wrapStartRoundOperation is a helper function to wrap start round operations.
func wrapStartRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (StartRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (StartRoundOperationResult, error) {
	if fn == nil {
		err := errors.New("operation function is nil")
		if logger != nil {
			logger.ErrorContext(ctx, "Operation function is nil", attr.String("operation", operationName))
		}
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "nil_function")
		}
		return StartRoundOperationResult{Error: err}, err
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
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(err), attr.String("operation", operationName))
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
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped), attr.String("operation", operationName))
		}
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return StartRoundOperationResult{Error: wrapped}, wrapped
	}

	if result.Error != nil {
		span.RecordError(result.Error)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Result error%s", operationName), attr.Error(result.Error), attr.String("operation", operationName))
		}
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "result_error")
		}
		return result, nil
	}

	if metrics != nil {
		metrics.RecordAPIRequest(ctx, operationName)
	}

	return result, nil
}
