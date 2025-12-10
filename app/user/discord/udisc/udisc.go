package udisc

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// UDiscManager defines the interface for UDisc name management operations.
type UDiscManager interface {
	HandleSetUDiscNameCommand(ctx context.Context, i *discordgo.InteractionCreate) (UDiscOperationResult, error)
}

// udiscManager implements the UDiscManager interface.
type udiscManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	config           *config.Config
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (UDiscOperationResult, error)) (UDiscOperationResult, error)
}

// NewUDiscManager creates a new UDiscManager instance.
func NewUDiscManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	cfg *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) UDiscManager {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("")
	}

	return &udiscManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		config:    cfg,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (UDiscOperationResult, error)) (UDiscOperationResult, error) {
			return operationWrapper(ctx, opName, fn, logger, tracer)
		},
	}
}

// UDiscOperationResult represents the result of a UDisc operation.
type UDiscOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// operationWrapper handles common logging and tracing for UDisc operations.
func operationWrapper(
	ctx context.Context,
	opName string,
	fn func(ctx context.Context) (UDiscOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
) (result UDiscOperationResult, err error) {
	ctx, span := tracer.Start(ctx, opName)
	defer span.End()

	logger.InfoContext(ctx, opName+" triggered",
		attr.String("operation", opName),
		attr.ExtractCorrelationID(ctx),
	)

	result, err = fn(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Error in "+opName,
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		span.RecordError(err)
		return result, err
	}

	logger.InfoContext(ctx, opName+" completed successfully",
		attr.String("operation", opName),
		attr.ExtractCorrelationID(ctx),
	)
	return result, nil
}
