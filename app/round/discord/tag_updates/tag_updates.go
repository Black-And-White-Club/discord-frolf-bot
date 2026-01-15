package tagupdates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discordgo "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TagUpdateManager defines the interface for tag update operations.
type TagUpdateManager interface {
	UpdateDiscordEmbedsWithTagChanges(ctx context.Context, payload roundevents.TagsUpdatedForScheduledRoundsPayloadV1, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (TagUpdateOperationResult, error)
	UpdateTagsInEmbed(ctx context.Context, channelID, messageID string, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (TagUpdateOperationResult, error)
}

// TagUpdateOperationResult represents the result of a tag update operation.
type TagUpdateOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

type tagUpdateManager struct {
	session             discordgo.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (TagUpdateOperationResult, error)) (TagUpdateOperationResult, error)
	guildConfigResolver guildconfig.GuildConfigResolver
}

// NewTagUpdateManager creates a new TagUpdateManager.
func NewTagUpdateManager(
	session discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
	guildConfigResolver guildconfig.GuildConfigResolver,
) TagUpdateManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating TagUpdateManager")
	}
	return &tagUpdateManager{
		session:   session,
		publisher: publisher,
		logger:    logger,
		helper:    helper,
		config:    config,
		tracer:    tracer,
		metrics:   metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (TagUpdateOperationResult, error)) (TagUpdateOperationResult, error) {
			return wrapTagUpdateOperation(ctx, opName, fn, logger, tracer, metrics)
		},
		guildConfigResolver: guildConfigResolver,
		guildConfigCache:    guildConfigCache,
		interactionStore:    interactionStore,
	}
}

// wrapTagUpdateOperation adds tracing, metrics, and error handling for operations.
func wrapTagUpdateOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (TagUpdateOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (TagUpdateOperationResult, error) {
	if fn == nil {
		return TagUpdateOperationResult{Error: errors.New("operation function is nil")}, nil
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
		return TagUpdateOperationResult{Error: wrapped}, wrapped
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
