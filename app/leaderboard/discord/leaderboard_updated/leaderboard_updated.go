package leaderboardupdated

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
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// LeaderboardUpdateManager defines the interface for leaderboard update operations.
type LeaderboardUpdateManager interface {
	HandleLeaderboardPagination(ctx context.Context, i *discordgo.InteractionCreate) (LeaderboardUpdateOperationResult, error)
	SendLeaderboardEmbed(ctx context.Context, channelID string, leaderboard []LeaderboardEntry, page int32) (LeaderboardUpdateOperationResult, error)
}

type leaderboardUpdateManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error)
}

// NewLeaderboardUpdateManager creates a new LeaderboardUpdateManager instance.
func NewLeaderboardUpdateManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) LeaderboardUpdateManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating LeaderboardUpdateManager")
	}
	return &leaderboardUpdateManager{
		session:             session,
		publisher:           publisher,
		logger:              logger,
		helper:              helper,
		config:              config,
		guildConfigResolver: guildConfigResolver,
		interactionStore:    interactionStore,
		guildConfigCache:    guildConfigCache,
		tracer:              tracer,
		metrics:             metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error)) (LeaderboardUpdateOperationResult, error) {
			return wrapLeaderboardUpdateOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

func wrapLeaderboardUpdateOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (LeaderboardUpdateOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (result LeaderboardUpdateOperationResult, err error) { // Named returns
	if fn == nil {
		return LeaderboardUpdateOperationResult{Error: errors.New("operation function is nil")}, nil
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
			result = LeaderboardUpdateOperationResult{Error: recoveredErr}
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
		return LeaderboardUpdateOperationResult{Error: wrapped}, wrapped
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

// LeaderboardUpdateOperationResult defines a standard result wrapper for leaderboard operations.
type LeaderboardUpdateOperationResult struct {
	Success any
	Failure any
	Error   error
}
