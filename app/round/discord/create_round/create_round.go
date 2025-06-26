package createround

import (
	"context"
	"encoding/json"
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
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// CreateRoundManager defines the interface for create round operations.
type CreateRoundManager interface {
	HandleCreateRoundCommand(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
	HandleCreateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
	UpdateInteractionResponse(ctx context.Context, correlationID, message string, edit ...*discordgo.WebhookEdit) (CreateRoundOperationResult, error)
	UpdateInteractionResponseWithRetryButton(ctx context.Context, correlationID, message string) (CreateRoundOperationResult, error)
	HandleCreateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
	SendRoundEventEmbed(channelID string, title roundtypes.Title, description roundtypes.Description, startTime sharedtypes.StartTime, location roundtypes.Location, creatorID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (CreateRoundOperationResult, error)
	SendCreateRoundModal(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
	HandleRetryCreateRound(ctx context.Context, i *discordgo.InteractionCreate) (CreateRoundOperationResult, error)
}

type createRoundManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error)
}

// NewCreateRoundManager creates a new CreateRoundManager instance.
func NewCreateRoundManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) CreateRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating CreateRoundManager")
	}
	return &createRoundManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
			return wrapCreateRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}
}

func wrapCreateRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (CreateRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (CreateRoundOperationResult, error) {
	if fn == nil {
		return CreateRoundOperationResult{Error: errors.New("operation function is nil")}, nil
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
		return CreateRoundOperationResult{Error: wrapped}, wrapped
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

type CreateRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// createEvent creates and marshals a Watermill message and assigns a correlation ID.
func (crm *createRoundManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, string, error) {
	correlationID := watermill.NewUUID()
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	if newEvent.Metadata == nil {
		newEvent.Metadata = make(map[string]string)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		crm.logger.ErrorContext(ctx, "Failed to marshal payload in createEvent", attr.Error(err))
		return nil, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	newEvent.Payload = payloadBytes
	newEvent.Metadata.Set("correlation_id", correlationID)
	newEvent.Metadata.Set("handler_name", "Create Round")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)
	newEvent.Metadata.Set("guild_id", crm.config.GetGuildID())

	return newEvent, correlationID, nil
}
