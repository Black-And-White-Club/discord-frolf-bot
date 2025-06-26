package scoreround

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ScoreRoundManager defines the interface for create round operations.
type ScoreRoundManager interface {
	HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error)
	HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error)
	SendScoreUpdateConfirmation(ctx context.Context, channelID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error)
	SendScoreUpdateError(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (ScoreRoundOperationResult, error)
	UpdateScoreEmbed(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (ScoreRoundOperationResult, error)
	AddLateParticipantToScorecard(ctx context.Context, channelID, messageID string, participants []roundtypes.Participant) (ScoreRoundOperationResult, error)
}

// scoreRoundManager implements the ScoreRoundManager interface.
type scoreRoundManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error)
}

// NewScoreRoundManager creates a new ScoreRoundManager instance.
func NewScoreRoundManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) ScoreRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating ScoreRoundManager",
			attr.Any("session", session),
			attr.Any("publisher", publisher),
			attr.Any("config", config),
		)
	}
	return &scoreRoundManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (ScoreRoundOperationResult, error)) (ScoreRoundOperationResult, error) {
			return wrapScoreRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

// wrapScoreRoundOperation adds tracing, metrics, and error handling for operations.
func wrapScoreRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (ScoreRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (ScoreRoundOperationResult, error) {
	if fn == nil {
		return ScoreRoundOperationResult{Error: errors.New("operation function is nil")}, nil
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
		return ScoreRoundOperationResult{Error: wrapped}, wrapped
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

// ScoreRoundOperationResult is the standard result container for operations.
type ScoreRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// userHasRole checks if a user has a specific Discord role
// func userHasRole(userRoles []string, requiredRoleID string) bool {
// 	for _, roleID := range userRoles {
// 		if roleID == requiredRoleID {
// 			return true
// 		}
// 	}
// 	return false
// }
