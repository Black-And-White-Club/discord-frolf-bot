package reset

import (
	"context"
	"fmt"
	"log/slog"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// ResetManager handles guild configuration reset operations.
type ResetManager interface {
	HandleResetCommand(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetConfirmButton(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetCancelButton(ctx context.Context, i *discordgo.InteractionCreate) error
}

type resetManager struct {
	session          discordgocommands.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
}

// NewResetManager creates a new reset manager.
func NewResetManager(
	session discordgocommands.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (ResetManager, error) {
	if session == nil {
		return nil, fmt.Errorf("session cannot be nil")
	}
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if helper == nil {
		return nil, fmt.Errorf("helper cannot be nil")
	}

	return &resetManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
	}, nil
}

// operationWrapper wraps reset operations with common error handling and logging.
func (rm *resetManager) operationWrapper(ctx context.Context, operationName string, fn func(context.Context) error) error {
	rm.logger.DebugContext(ctx, fmt.Sprintf("Starting %s operation", operationName))

	err := fn(ctx)
	if err != nil {
		rm.logger.ErrorContext(ctx, fmt.Sprintf("%s operation failed", operationName),
			attr.Error(err))
		return fmt.Errorf("%s failed: %w", operationName, err)
	}

	rm.logger.DebugContext(ctx, fmt.Sprintf("%s operation completed successfully", operationName))
	return nil
}
