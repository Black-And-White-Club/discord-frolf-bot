package updateround

import (
	"context"
	"encoding/json"
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
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// UpdateRoundManager defines the interface for update round operations.
type UpdateRoundManager interface {
	UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID string, title *roundtypes.Title, description *roundtypes.Description, startTime *sharedtypes.StartTime, location *roundtypes.Location) (UpdateRoundOperationResult, error)
	HandleEditRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error)
	SendUpdateRoundModal(ctx context.Context, i *discordgo.InteractionCreate, roundID sharedtypes.RoundID) (UpdateRoundOperationResult, error)
	HandleUpdateRoundModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error)
	HandleUpdateRoundModalCancel(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error)
}

type updateRoundManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error)
	guildConfigResolver guildconfig.GuildConfigResolver
}

// NewUpdateRoundManager creates a new UpdateRoundManager instance.
func NewUpdateRoundManager(
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
) UpdateRoundManager {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating UpdateRoundManager")
	}
	return &updateRoundManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (UpdateRoundOperationResult, error)) (UpdateRoundOperationResult, error) {
			return wrapUpdateRoundOperation(ctx, opName, fn, logger, tracer, metrics)
		},
		guildConfigResolver: guildConfigResolver,
		guildConfigCache:    guildConfigCache,
	}
}

func wrapUpdateRoundOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (UpdateRoundOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (result UpdateRoundOperationResult, err error) {
	if fn == nil {
		result = UpdateRoundOperationResult{Error: errors.New("operation function is nil")}
		return result, nil
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
			panicErr := fmt.Errorf("panic in %s: %v", operationName, r)
			span.RecordError(panicErr)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(panicErr))
			}
			if metrics != nil {
				metrics.RecordAPIError(ctx, operationName, "panic")
			}
			result = UpdateRoundOperationResult{Error: panicErr}
			err = panicErr
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
		return UpdateRoundOperationResult{Error: wrapped}, wrapped
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

type UpdateRoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// createEvent creates and marshals a Watermill message.
func (crm *updateRoundManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	if newEvent.Metadata == nil {
		newEvent.Metadata = make(map[string]string)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		crm.logger.ErrorContext(ctx, "Failed to marshal payload in createEvent", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	newEvent.Payload = payloadBytes
	newEvent.Metadata.Set("handler_name", "Update Round")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)

	// Use GuildID from the actual interaction instead of config
	if i.Interaction.GuildID != "" {
		newEvent.Metadata.Set("guild_id", i.Interaction.GuildID)
	} else {
		crm.logger.WarnContext(ctx, "Interaction missing GuildID in updateRoundManager.createEvent, but no longer falling back to global config")
	}

	return newEvent, nil
}
