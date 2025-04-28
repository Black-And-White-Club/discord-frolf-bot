package deleteround

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
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// DeleteRoundManager defines the interface for delete round operations.
type DeleteRoundManager interface {
	HandleDeleteRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (DeleteRoundOperationResult, error)
	DeleteRoundEventEmbed(ctx context.Context, eventMessageID sharedtypes.RoundID, channelID string) (DeleteRoundOperationResult, error)
}

type deleteRoundManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (DeleteRoundOperationResult, error)) (DeleteRoundOperationResult, error)
}

// NewDeleteRoundManager creates a new DeleteRoundManager instance.
func NewDeleteRoundManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) DeleteRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating DeleteRoundManager")
	}
	return &deleteRoundManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (DeleteRoundOperationResult, error)) (DeleteRoundOperationResult, error) {
			return wrapDeleteRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

func wrapDeleteRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (DeleteRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (DeleteRoundOperationResult, error) {
	if fn == nil {
		return DeleteRoundOperationResult{Error: errors.New("operation function is nil")}, nil
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
		return DeleteRoundOperationResult{Error: wrapped}, wrapped
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

type DeleteRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// // createEvent creates and marshals a Watermill message.
// func (drm *deleteRoundManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
// 	newEvent := message.NewMessage(watermill.NewUUID(), nil)

// 	if newEvent.Metadata == nil {
// 		newEvent.Metadata = make(map[string]string)
// 	}

// 	payloadBytes, err := json.Marshal(payload)
// 	if err != nil {
// 		drm.logger.ErrorContext(ctx, "Failed to marshal payload in createEvent", attr.Error(err))
// 		return nil, fmt.Errorf("failed to marshal payload: %w", err)
// 	}

// 	newEvent.Payload = payloadBytes
// 	newEvent.Metadata.Set("handler_name", "Delete Round")
// 	newEvent.Metadata.Set("topic", topic)
// 	newEvent.Metadata.Set("domain", "discord")
// 	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
// 	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
// 	newEvent.Metadata.Set("guild_id", drm.config.Discord.GuildID)

// 	return newEvent, nil
// }
