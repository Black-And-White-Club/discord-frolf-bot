package roundrsvp

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
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// RoundRsvpManager defines the interface for round RSVP operations.
type RoundRsvpManager interface {
	HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error)
	UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID string, acceptedParticipants, declinedParticipants, tentativeParticipants []roundtypes.Participant) (RoundRsvpOperationResult, error)
	InteractionJoinRoundLate(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error)
}

type roundRsvpManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	interactionStore    storage.ISInterface
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (RoundRsvpOperationResult, error)) (RoundRsvpOperationResult, error)
	guildConfigResolver guildconfig.GuildConfigResolver
}

// NewRoundRsvpManager creates a new RoundRsvpManager instance.
func NewRoundRsvpManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) RoundRsvpManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating RoundRsvpManager")
	}
	return &roundRsvpManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (RoundRsvpOperationResult, error)) (RoundRsvpOperationResult, error) {
			return wrapRoundRsvpOperation(ctx, opName, fn, logger, tracer, metrics)
		},
		guildConfigResolver: guildConfigResolver,
	}
}

// wrapRoundRsvpOperation adds tracing, metrics, and error handling for operations.
func wrapRoundRsvpOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (RoundRsvpOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (RoundRsvpOperationResult, error) {
	if fn == nil {
		return RoundRsvpOperationResult{Error: errors.New("operation function is nil")}, nil
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
		return RoundRsvpOperationResult{Error: wrapped}, wrapped
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

// RoundRsvpOperationResult is the standard result container for operations.
type RoundRsvpOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
