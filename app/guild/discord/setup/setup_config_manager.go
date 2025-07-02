package setup

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// SetupManager interface defines the contract for guild setup management
type SetupManager interface {
	HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error
	SendSetupModal(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleSetupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) error
}

type setupManager struct {
	session          *discordgo.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, opName string, fn func(ctx context.Context) error) error
}

// NewSetupManager creates a new SetupManager instance
func NewSetupManager(
	session *discordgo.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (SetupManager, error) {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating Guild SetupManager")
	}

	return &setupManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) error) error {
			return wrapSetupOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}, nil
}

// wrapSetupOperation wraps setup operations with observability and error handling
func wrapSetupOperation(
	ctx context.Context,
	opName string,
	fn func(ctx context.Context) error,
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) error {
	// Start span for tracing
	ctx, span := tracer.Start(ctx, fmt.Sprintf("guild.setup.%s", opName))
	defer span.End()

	start := time.Now()

	// Execute the operation
	err := fn(ctx)

	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		if logger != nil {
			logger.ErrorContext(ctx, "Guild setup operation failed",
				"operation", opName,
				"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()),
				"error", err)
		}
	} else {
		if logger != nil {
			logger.InfoContext(ctx, "Guild setup operation completed",
				"operation", opName,
				"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()))
		}
	}

	return err
}
